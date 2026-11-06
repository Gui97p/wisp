#!/usr/bin/env python3
import argparse
import datetime
import json
import math
import os
import platform
import shutil
import statistics
import subprocess
import sys
import tempfile
import threading
import time
from pathlib import Path

HERE = Path(__file__).resolve().parent
ROOT = HERE.parent
PROGRAMS = HERE / "programs"
BUILD = HERE / "build"
RESULTS = HERE / "results"

WISP = Path(os.environ.get("WISP", ROOT / "bin" / "wisp"))
CC = os.environ.get("CC", "cc")


class Variant:
    def __init__(self, name, language, source, tool, command, output="stdout"):
        self.name = name
        self.language = language
        self.source = source
        self.tool = tool
        self.command = command
        self.output = output

    def available(self):
        if self.tool == "wisp":
            return WISP.exists()
        return shutil.which(self.tool) is not None


def wisp_command(src, out, work):
    return [str(WISP), "build", "--bin", str(out), "--obj", str(work / "obj"), "--asm", str(work / "asm")], src


def c_command(level):
    def build(src, out, work):
        return [CC, level, "-o", str(out), str(src / "main.c")], None
    return build


def go_command(src, out, work):
    return ["go", "build", "-o", str(out), "main.go"], src


def zig_command(mode):
    def build(src, out, work):
        return ["zig", "build-exe", str(src / "main.zig"), "-O", mode, f"-femit-bin={out}"], work
    return build


def rust_command(level):
    def build(src, out, work):
        return ["rustc", "-C", f"opt-level={level}", "-o", str(out), str(src / "main.rs")], None
    return build


VARIANTS = [
    Variant("wisp", "wisp", "main.wsp", "wisp", wisp_command),
    Variant("c-O0", "c", "main.c", CC, c_command("-O0")),
    Variant("c-O2", "c", "main.c", CC, c_command("-O2")),
    Variant("go", "go", "main.go", "go", go_command),
    Variant("zig-debug", "zig", "main.zig", "zig", zig_command("Debug"), output="stderr"),
    Variant("zig-fast", "zig", "main.zig", "zig", zig_command("ReleaseFast"), output="stderr"),
    Variant("rust-O0", "rust", "main.rs", "rustc", rust_command("0")),
    Variant("rust-O2", "rust", "main.rs", "rustc", rust_command("2")),
]


class Sample:
    def __init__(self, wall, user, system, rss, out, err, code):
        self.wall = wall
        self.user = user
        self.system = system
        self.rss = rss
        self.out = out
        self.err = err
        self.code = code


def run_once(binary, timeout):
    with tempfile.TemporaryFile() as out, tempfile.TemporaryFile() as err:
        start = time.perf_counter()
        proc = subprocess.Popen([str(binary)], stdout=out, stderr=err)
        timer = threading.Timer(timeout, proc.kill)
        timer.start()
        _, status, usage = os.wait4(proc.pid, 0)
        wall = time.perf_counter() - start
        timer.cancel()
        proc.returncode = os.waitstatus_to_exitcode(status)
        out.seek(0)
        err.seek(0)
        return Sample(
            wall,
            usage.ru_utime,
            usage.ru_stime,
            usage.ru_maxrss,
            out.read().decode(errors="replace"),
            err.read().decode(errors="replace"),
            proc.returncode,
        )


def peak_rss_kb(binary, timeout):
    timer_tool = shutil.which("time") and Path("/usr/bin/time")
    if not timer_tool or not timer_tool.exists():
        return None
    with tempfile.NamedTemporaryFile("r") as report:
        try:
            subprocess.run(
                [str(timer_tool), "-f", "%M", "-o", report.name, str(binary)],
                stdout=subprocess.DEVNULL,
                stderr=subprocess.DEVNULL,
                timeout=timeout,
            )
            return int(report.read().strip().splitlines()[-1])
        except (ValueError, IndexError, subprocess.SubprocessError, OSError):
            return None


def tool_version(tool):
    if tool == "wisp":
        cmd = [str(WISP), "version", "--short"]
    elif tool == "go":
        cmd = ["go", "version"]
    else:
        cmd = [tool, "--version"]
    try:
        result = subprocess.run(cmd, capture_output=True, text=True, timeout=20)
        text = (result.stdout or result.stderr).strip().splitlines()
        return text[0] if text else ""
    except (OSError, subprocess.SubprocessError):
        return ""


def git_state():
    try:
        rev = subprocess.run(["git", "rev-parse", "--short", "HEAD"], cwd=ROOT, capture_output=True, text=True).stdout.strip()
        dirty = bool(subprocess.run(["git", "status", "--porcelain"], cwd=ROOT, capture_output=True, text=True).stdout.strip())
        return rev, dirty
    except OSError:
        return "", False


def cpu_model():
    try:
        for line in Path("/proc/cpuinfo").read_text().splitlines():
            if line.startswith("model name"):
                return line.split(":", 1)[1].strip()
    except OSError:
        pass
    return platform.processor()


def stats(values):
    ordered = sorted(values)
    mean = statistics.fmean(ordered)
    stdev = statistics.stdev(ordered) if len(ordered) > 1 else 0.0
    return {
        "min": ordered[0],
        "median": statistics.median(ordered),
        "mean": mean,
        "stdev": stdev,
        "max": ordered[-1],
        "cv": (stdev / mean * 100.0) if mean else 0.0,
    }


def build_variant(variant, bench, out, work):
    src = PROGRAMS / bench
    work.mkdir(parents=True, exist_ok=True)
    command, cwd = variant.command(src, out, work)
    if cwd is None:
        cwd = src
    start = time.perf_counter()
    result = subprocess.run(command, cwd=cwd, capture_output=True, text=True)
    elapsed = time.perf_counter() - start
    if result.returncode != 0 or not out.exists():
        detail = (result.stderr or result.stdout).strip().splitlines()
        return None, elapsed, " | ".join(detail[-3:])
    return out, elapsed, ""


def measure(variant, bench, args, expected):
    work = BUILD / bench / variant.name
    if work.exists():
        shutil.rmtree(work)
    out = work / "program"
    binary, compile_time, error = build_variant(variant, bench, out, work)
    if binary is None:
        return {"status": "build failed", "error": error, "compile_s": compile_time}

    print(f"  {variant.name:<10} built in {compile_time:.2f}s, running", file=sys.stderr)
    for _ in range(args.warmup):
        run_once(binary, args.timeout)

    samples = []
    for _ in range(args.runs):
        sample = run_once(binary, args.timeout)
        samples.append(sample)
        if sample.code != 0:
            return {"status": f"exit {sample.code}", "error": sample.err.strip()[-200:], "compile_s": compile_time}

    printed = samples[0].out.strip() if variant.output == "stdout" else samples[0].err.strip()
    verified = printed == expected if expected is not None else None
    walls = [s.wall for s in samples]
    return {
        "status": "ok" if verified is not False else "wrong output",
        "output": printed,
        "verified": verified,
        "compile_s": compile_time,
        "size": binary.stat().st_size,
        "wall": walls,
        "stats": stats(walls),
        "cpu": statistics.median([s.user + s.system for s in samples]),
        "rss_kb": peak_rss_kb(binary, args.timeout),
    }


def fmt_ms(seconds):
    return f"{seconds * 1000:9.1f}"


def rss_text(kb):
    return f"{kb / 1024:.1f}" if kb is not None else "-"


def ratio(entry, ref):
    if not entry or not ref or entry.get("status") != "ok" or ref.get("status") != "ok":
        return None
    return entry["stats"]["median"] / ref["stats"]["median"]


def fmt_ratio(value):
    return f"{value:8.2f}x" if value is not None else f"{'-':>9}"


def print_benchmark(bench, results, order, ref, ref2):
    expected = results.get("expected")
    print()
    print(f"== {bench}   expected: {expected}")
    head = f"{'variant':<11}{'build s':>8}{'size KB':>9}{'min ms':>10}{'median ms':>11}{'stdev %':>9}{'cpu ms':>9}{'rss MB':>8}  vs {ref:<7} vs {ref2:<7}"
    print(head)
    print("-" * len(head))
    rows = results["variants"]
    for name in order:
        entry = rows.get(name)
        if entry is None:
            continue
        if entry["status"] not in ("ok", "wrong output"):
            print(f"{name:<11}{entry['compile_s']:>8.2f}  {entry['status']}: {entry.get('error', '')[:60]}")
            continue
        s = entry["stats"]
        mark = "" if entry["status"] == "ok" else "  WRONG OUTPUT: " + entry["output"]
        print(
            f"{name:<11}{entry['compile_s']:>8.2f}{entry['size'] / 1024:>9.0f}{fmt_ms(s['min']):>10}{fmt_ms(s['median']):>11}"
            f"{s['cv']:>9.1f}{fmt_ms(entry['cpu']):>9}{rss_text(entry['rss_kb']):>8}  "
            f"{fmt_ratio(ratio(entry, rows.get(ref)))}  {fmt_ratio(ratio(entry, rows.get(ref2)))}{mark}"
        )


def geomean(values):
    values = [v for v in values if v is not None and v > 0]
    if not values:
        return None
    return math.exp(sum(math.log(v) for v in values) / len(values))


def print_summary(data, order, ref, ref2):
    print()
    print("== summary (geometric mean of median time ratios)")
    head = f"{'variant':<11}{'vs ' + ref:>12}{'vs ' + ref2:>12}{'benchmarks':>12}{'total ms':>12}"
    print(head)
    print("-" * len(head))
    for name in order:
        first, second, count, total = [], [], 0, 0.0
        for bench in data.values():
            rows = bench["variants"]
            entry = rows.get(name)
            if not entry or entry["status"] != "ok":
                continue
            count += 1
            total += entry["stats"]["median"]
            first.append(ratio(entry, rows.get(ref)))
            second.append(ratio(entry, rows.get(ref2)))
        if count == 0:
            continue
        print(f"{name:<11}{fmt_ratio(geomean(first)):>12}{fmt_ratio(geomean(second)):>12}{count:>12}{fmt_ms(total):>12}")


def print_comparison(data, old, order):
    print()
    print(f"== comparison with {old['meta'].get('label') or old['meta']['date']} (rev {old['meta'].get('git_rev', '?')})")
    head = f"{'benchmark':<12}{'variant':<11}{'before ms':>11}{'after ms':>11}{'change':>10}"
    print(head)
    print("-" * len(head))
    deltas = {}
    for bench, now in data.items():
        before = old["results"].get(bench, {}).get("variants", {})
        for name in order:
            a, b = before.get(name), now["variants"].get(name)
            if not a or not b or a.get("status") != "ok" or b.get("status") != "ok":
                continue
            old_m, new_m = a["stats"]["median"], b["stats"]["median"]
            change = new_m / old_m
            deltas.setdefault(name, []).append(change)
            print(f"{bench:<12}{name:<11}{fmt_ms(old_m):>11}{fmt_ms(new_m):>11}{(change - 1) * 100:>+9.1f}%")
    print()
    for name, values in deltas.items():
        g = geomean(values)
        print(f"  {name:<11} geometric mean change: {(g - 1) * 100:+.1f}%")


def load_comparison(spec):
    if spec == "last":
        path = RESULTS / "latest.json"
    else:
        path = Path(spec)
        if not path.exists():
            path = RESULTS / spec
    if not path.exists():
        print(f"no results to compare with ({path})", file=sys.stderr)
        return None
    return json.loads(path.read_text())


def ensure_compiler():
    print("building the Wisp compiler", file=sys.stderr)
    result = subprocess.run(["go", "build", "-o", str(WISP), "./cmd/main"], cwd=ROOT, capture_output=True, text=True)
    if result.returncode != 0:
        print(result.stderr, file=sys.stderr)
        sys.exit(1)


def main():
    parser = argparse.ArgumentParser(description="Compare Wisp with C, Go, Zig and Rust on the same programs")
    parser.add_argument("--only", help="comma separated benchmarks")
    parser.add_argument("--variants", "--langs", dest="variants", help="comma separated variants, or a language (c, go, ...)")
    parser.add_argument("--runs", type=int, default=5)
    parser.add_argument("--warmup", type=int, default=1)
    parser.add_argument("--quick", action="store_true", help="3 runs, no warmup")
    parser.add_argument("--timeout", type=float, default=180.0)
    parser.add_argument("--ref", default="c-O0")
    parser.add_argument("--ref2", default="c-O2")
    parser.add_argument("--compare", metavar="FILE|last", help="compare with a saved result")
    parser.add_argument("--label", default="", help="name for this run, kept in the saved file")
    parser.add_argument("--no-save", action="store_true")
    parser.add_argument("--no-compiler-build", action="store_true")
    parser.add_argument("--list", action="store_true")
    args = parser.parse_args()

    if args.quick:
        args.runs, args.warmup = 3, 0

    benchmarks = sorted(p.name for p in PROGRAMS.iterdir() if p.is_dir())
    if args.only:
        wanted = args.only.split(",")
        missing = [b for b in wanted if b not in benchmarks]
        if missing:
            sys.exit(f"unknown benchmark: {', '.join(missing)} (have: {', '.join(benchmarks)})")
        benchmarks = [b for b in benchmarks if b in wanted]

    variants = VARIANTS
    if args.variants:
        wanted = args.variants.split(",")
        variants = [v for v in VARIANTS if v.name in wanted or v.language in wanted]
        if not variants:
            sys.exit(f"no variant matches {args.variants} (have: {', '.join(v.name for v in VARIANTS)})")

    if args.list:
        print("benchmarks:", ", ".join(benchmarks))
        for v in VARIANTS:
            print(f"  {v.name:<10} {'available' if v.available() else 'missing ' + v.tool}")
        return

    if not args.no_compiler_build and any(v.tool == "wisp" for v in variants):
        ensure_compiler()

    usable = [v for v in variants if v.available()]
    skipped = [v.name for v in variants if not v.available()]
    if skipped:
        print(f"skipping (tool not installed): {', '.join(skipped)}", file=sys.stderr)

    previous = load_comparison(args.compare) if args.compare else None

    data = {}
    for bench in benchmarks:
        print(f"{bench}", file=sys.stderr)
        expected_file = PROGRAMS / bench / "expected.txt"
        expected = expected_file.read_text().strip() if expected_file.exists() else None
        data[bench] = {"expected": expected, "variants": {}}
        for variant in usable:
            if not (PROGRAMS / bench / variant.source).exists():
                continue
            data[bench]["variants"][variant.name] = measure(variant, bench, args, expected)

    order = [v.name for v in usable]
    for bench in benchmarks:
        print_benchmark(bench, data[bench], order, args.ref, args.ref2)
    print_summary(data, order, args.ref, args.ref2)
    if previous:
        print_comparison(data, previous, order)

    if not args.no_save:
        rev, dirty = git_state()
        now = datetime.datetime.now()
        meta = {
            "date": now.isoformat(timespec="seconds"),
            "label": args.label,
            "git_rev": rev,
            "git_dirty": dirty,
            "machine": f"{platform.system()} {platform.release()} {platform.machine()}",
            "cpu": cpu_model(),
            "runs": args.runs,
            "warmup": args.warmup,
            "tools": {v.name: tool_version(v.tool) for v in usable},
        }
        RESULTS.mkdir(exist_ok=True)
        stamp = now.strftime("%Y%m%d-%H%M%S") + (f"-{args.label}" if args.label else "")
        path = RESULTS / f"{stamp}.json"
        payload = json.dumps({"meta": meta, "results": data}, indent=2)
        path.write_text(payload)
        (RESULTS / "latest.json").write_text(payload)
        print(f"\nsaved {path.relative_to(ROOT)}")


if __name__ == "__main__":
    main()

import * as fs from "fs";
import * as path from "path";
import * as vscode from "vscode";
import {
    LanguageClient,
    LanguageClientOptions,
    ServerOptions,
} from "vscode-languageclient/node";

let client: LanguageClient | undefined;

const BINARY_NAME = process.platform === "win32" ? "wisp.exe" : "wisp";

function isExecutable(file: string): boolean {
    try {
        fs.accessSync(file, fs.constants.X_OK);
        return fs.statSync(file).isFile();
    } catch {
        return false;
    }
}

function findOnPath(): string | undefined {
    const dirs = (process.env.PATH ?? "").split(path.delimiter);
    for (const dir of dirs) {
        const candidate = path.join(dir, BINARY_NAME);
        if (isExecutable(candidate)) {
            return candidate;
        }
    }
    return undefined;
}

function resolveServerPath(): string {
    const config = vscode.workspace.getConfiguration("wisp");
    const inspected = config.inspect<string>("serverPath");
    const explicit =
        inspected?.workspaceFolderValue ??
        inspected?.workspaceValue ??
        inspected?.globalValue;

    if (explicit) {
        return explicit;
    }

    return findOnPath() ?? "wisp";
}

export function activate(context: vscode.ExtensionContext) {
    const serverOptions: ServerOptions = {
        command: resolveServerPath(),
        args: ["lsp"],
    };

    const clientOptions: LanguageClientOptions = {
        documentSelector: [{ scheme: "file", language: "wisp" }],
    };

    client = new LanguageClient(
        "wisp",
        "Wisp Language Server",
        serverOptions,
        clientOptions
    );

    client.start();
}

export function deactivate(): Thenable<void> | undefined {
    if (!client) {
        return undefined;
    }
    return client.stop();
}

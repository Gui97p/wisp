global __wisp_emit
global __wisp_error_emit
global __wisp_panic_at
global __wisp_panic
global __wisp_panic_site
%ifndef WISP_LIBRARY
extern __wisp_main
%endif
global __wisp_envp
global __wisp_args_ptr
global __wisp_args_len

%if WISP_TRACE
global __wisp_site
global __wisp_trace
%endif

global __wisp_after_main
global __wisp_exit_hook

%ifndef WISP_LIBRARY
global main
%endif

%macro lit 1
	section .rodata
%%text: db %1
%%end:
	section .text
	lea rsi, [rel %%text]
	mov edx, %%end - %%text
	call __wisp_write_error
%endmacro

section .rodata

section .bss
__wisp_altstack: resb 16384
__wisp_envp: resq 1
__wisp_args_ptr: resq 1
__wisp_args_len: resq 1
__wisp_exit_hook: resq 1

section .text
__wisp_emit:
	mov rdx, rsi
	mov rsi, rdi
	mov edi, 1
	mov eax, 1
	syscall

	push 10
	mov rsi, rsp
	mov edx, 1
	mov edi, 1
	mov eax, 1
	syscall

	pop rax
	ret

__wisp_error_emit:
	push rbx
	mov rbx, rdi
	sub rsp, 32

	mov rdx, [rbx+72]
	test rdx, rdx
	jz .fallback

	mov rsi, [rbx+64]
	call __wisp_write_error
	jmp .end

.fallback:
	lit "error["
	mov rsi, [rbx]
	mov rdx, [rbx+8]
	call __wisp_write_error

	lit "]: "
	mov rsi, [rbx+24]
	mov rdx, [rbx+32]
	call __wisp_write_error

	lit `\n--> `
	mov rsi, [rbx+40]
	mov rdx, [rbx+48]
	call __wisp_write_error

	lit ":"
	mov rdi, [rbx+56]
	lea rsi, [rsp+32]
	call __wisp_utoa

	mov rsi, rax
	call __wisp_write_error

.end:
	lit `\n`
	add rsp, 32
	pop rbx
	ret

__wisp_write_error:
	mov rax, 1
	mov rdi, 2
	syscall
	ret

__wisp_panic_at:
	push rdx
	push rcx
	push rdi
	push rsi
	lit "panic: "

	pop rdx
	pop rsi
	call __wisp_write_error

	push 10
	mov rsi, rsp
	mov edx, 1
	call __wisp_write_error
	add rsp, 8

	pop rdx
	pop rsi
	call __wisp_write_error
	
%if WISP_TRACE
	call __wisp_trace
%endif

	and rsp, -16
	call __wisp_run_exit_hook
	mov edi, 1
	mov eax, 60
	syscall

__wisp_write_cstr:
	xor edx, edx
.length:
	cmp byte [rsi + rdx], 0
	je .write
	inc rdx
	jmp .length
.write:
	jmp __wisp_write_error

__wisp_write_number:
	sub rsp, 32
	lea rsi, [rsp + 32]
	call __wisp_utoa
	mov rsi, rax
	call __wisp_write_error
	add rsp, 32
	ret

__wisp_panic_site:
	push rdx
	push rsi
	push rdi
	lit "panic: "
	pop rsi
	call __wisp_write_cstr
	lit `\n--> `
	pop rsi
	call __wisp_write_cstr
	pop rbx
	lit ":"
	mov rdi, rbx
	shr rdi, 8
	call __wisp_write_number
	lit ":"
	movzx edi, bl
	call __wisp_write_number
	lit `\n`

%if WISP_TRACE
	call __wisp_trace
%endif

	mov edi, 1
	mov eax, 60
	syscall

__wisp_install_signals:
	push rdi
	push rsi
	push rdx
	sub rsp, 32
	lea rax, [rel __wisp_altstack]
	mov [rsp], rax
	mov qword [rsp+8], 0
	mov qword [rsp+16], 16384
	mov rdi, rsp
	xor esi, esi
	mov eax, 131
	syscall

	lea rax, [rel __wisp_segv]
	mov [rsp], rax
	mov qword [rsp+8], 0x0C000004
	lea rax, [rel __wisp_restore]
	mov [rsp+16], rax
	mov qword [rsp+24], 0
	mov edi, 11
	mov rsi, rsp
	xor edx, edx
	mov r10d, 8
	mov eax, 13
	syscall
	add rsp, 32
	pop rdx
	pop rsi
	pop rdi
	ret

__wisp_restore:
	mov eax, 15
	syscall

__wisp_segv:
	mov rax, [rsi+16]
	mov rcx, [rdx+160]
	mov rdx, rcx
	sub rdx, 65536
	cmp rax, rdx
	jb .other
	add rcx, 16777216
	cmp rax, rcx
	jae .other
	lit {"panic: stack overflow", 10}
	jmp .done
.other:
	lit {"panic: invalid memory access", 10}
.done:
	mov edi, 1
	mov eax, 60
	syscall

__wisp_panic:
	push rdi
	push rsi
	lit "panic: "

	pop rdx
	pop rsi
	call __wisp_write_error

	push 10
	mov rsi, rsp
	mov edx, 1
	call __wisp_write_error

%if WISP_TRACE
	call __wisp_trace
%endif

	and rsp, -16
	call __wisp_run_exit_hook
	mov edi, 1
	mov eax, 60
	syscall

__wisp_utoa:
	mov rax, rdi
	mov rcx, 10
	mov r8, rsi
.loop:
	xor edx, edx
	div rcx

	add dl, '0'
	dec rsi
	mov [rsi], dl

	test rax, rax
	jnz .loop

	mov rax, rsi
	mov rdx, r8
	sub rdx, rsi

	ret

__wisp_run_exit_hook:
	push rax
	push rbp
	mov rax, [rel __wisp_exit_hook]
	test rax, rax
	jz .nohook
	xor ebp, ebp
	call rax
.nohook:
	pop rbp
	pop rax
	ret

%if WISP_TRACE
__wisp_site:
	mov rax, [rdi+8]
	lea rcx, [rel __wisp_after_main]
	cmp rax, rcx
	je .end
	mov rcx, [rdi]
	test rcx, rcx
	jz .end
	mov rcx, [rcx-8]
	mov rdx, 0x57495350
	cmp [rcx], rdx
	jne .end
	sub rax, [rcx+8]
	mov rdx, [rcx+16]
	lea rsi, [rcx+24]
.search:
	test rdx, rdx
	jz .none
	cmp [rsi], rax
	je .found
	add rsi, 24
	dec rdx
	jmp .search
.found:
	mov rax, [rsi+8]
	mov rdx, [rsi+16]
	ret
.none:
	xor eax, eax
	xor edx, edx
	ret
.end:
	xor eax, eax
	mov rdx, -1
	ret

__wisp_trace:
	push r12
	push r13
	push r14 ; n
	xor r14d, r14d ; n
	mov r12, rbp
	xor r13d, r13d

	lit `\n`

.next:
	mov rdi, r12
	call __wisp_site
	cmp rdx, -1
	je .done
	mov r12, [r12]
	test rax, rax
	jz .next
	test r14, r14
	jnz .print
	cmp rdx, 4
	jb .user
	cmp dword [rax], 0x2f647473
	je .next

.user:
	mov r14d, 1
.print:
	mov rsi, rax
	call __wisp_write_error
	inc r13
	cmp r13, 16
	jne .next

	lit `... more frames\n`

.done:
	pop r14
	pop r13
	pop r12
	ret
%endif

%ifdef WISP_LIBRARY
__wisp_after_main:
	ret
%else
main:
	push rbp
	mov rbp, rsp
	mov [rel __wisp_envp], rdx
	call __wisp_install_signals

	mov r9, rdi
	imul r9, 16
	sub rsp, r9
	xor r10d, r10d
.loop:
	cmp r10, rdi
	je .endloop
	mov rcx, [rsi+r10*8]

	xor edx, edx
.loopb:
	cmp byte [rcx+rdx], 0
	je .endloopb
	inc rdx
	jmp .loopb
.endloopb:
	mov r11, r10
	shl r11, 4
	mov [rsp+r11], rcx
	mov [rsp+r11+8], rdx
	inc r10
	jmp .loop

.endloop:
	mov [rel __wisp_args_ptr], rsp
	mov [rel __wisp_args_len], rdi
	call __wisp_main
__wisp_after_main:
	and rsp, -16
	call __wisp_run_exit_hook
	leave
	ret
%endif

section .note.GNU-stack noalloc noexec nowrite progbits

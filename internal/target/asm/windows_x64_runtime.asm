default rel

global __wisp_emit
global __wisp_error_emit
global __wisp_panic_at
global __wisp_panic
extern __wisp_main
global __wisp_envp
global __wisp_args_ptr
global __wisp_args_len

%if WISP_TRACE
global __wisp_site
global __wisp_trace
%endif

global __wisp_after_main
global __wisp_exit_hook

global main

extern GetStdHandle
extern WriteFile
extern ExitProcess

%macro lit 1
	section .rodata
%%text: db %1
%%end:
	section .text
	lea rsi, [rel %%text]
	mov edx, %%end - %%text
	call __wisp_write_error
%endmacro

section .rodata rdata align=16

section .bss bss align=8
__wisp_envp: resq 1
__wisp_args_ptr: resq 1
__wisp_args_len: resq 1
__wisp_exit_hook: resq 1

section .text code align=16
__wisp_emit:
	mov rdx, rsi
	mov rsi, rdi
	mov ecx, -11
	call __wisp_write_to

	push 10
	mov rsi, rsp
	mov edx, 1
	mov ecx, -11
	call __wisp_write_to

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
	mov ecx, -12

__wisp_write_to:
	push rdx
	push r8
	push r9
	push r10
	mov rdi, rsp
	and rsp, -16
	sub rsp, 64
	mov [rsp+48], rdx
	call GetStdHandle
	mov rcx, rax
	mov rdx, rsi
	mov r8, [rsp+48]
	lea r9, [rsp+40]
	mov qword [rsp+32], 0
	call WriteFile
	mov rsp, rdi
	pop r10
	pop r9
	pop r8
	pop rdx
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
	sub rsp, 32
	mov ecx, 1
	call ExitProcess

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
	sub rsp, 32
	mov ecx, 1
	call ExitProcess

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
        mov rsi, rax
        call __wisp_write_error
        inc r13
        cmp r13, 16
        jne .next

        lit `... more frames\n`

.done:
        pop r13
        pop r12
        ret
%endif

main:
	push rbp
	mov rbp, rsp
	mov [rel __wisp_envp], rdx

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
	push rax
	push rax
	mov rax, [rel __wisp_exit_hook]
	test rax, rax
	jz .nohook
	call rax
.nohook:
	pop rax
	pop rax
	leave
	ret


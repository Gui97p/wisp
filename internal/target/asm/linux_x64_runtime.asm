global __wisp_emit
global __wisp_error_emit
global __wisp_panic

%macro lit 1
	section .rodata
%%text: db %1
%%end:
	section .text
	lea rsi, [rel %%text]
	mov edx, %%end - %%text
	call __wisp_write_error
%endmacro

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

	lit "error["
	mov rsi, [rbx]
	mov rdx, [rbx+8]
	call __wisp_write_error

	lit "]: "
	mov rsi, [rbx+24]
	mov rdx, [rbx+32]
	call __wisp_write_error

	lit `\n  at `
	mov rsi, [rbx+40]
	mov rdx, [rbx+48]
	call __wisp_write_error

	lit ":"
	mov rdi, [rbx+56]
	lea rsi, [rsp+32]
	call __wisp_utoa

	mov rsi, rax
	call __wisp_write_error

	lit `\n`
	add rsp, 32
	pop rbx
	ret

__wisp_write_error:
	mov rax, 1
	mov rdi, 2
	syscall
	ret

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

section .note.GNU-stack noalloc noexec nowrite progbits

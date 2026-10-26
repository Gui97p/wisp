global _start
extern main

section .text
_start:
	xor ebp, ebp
	mov edi, [rsp]
	lea rsi, [rsp+8]
	call main
	mov rdi, rax
	mov eax, 60
	syscall

section .note.GNU-stack noalloc noexec nowrite progbits

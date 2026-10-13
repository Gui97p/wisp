global _start
extern main

section .text
_start:
	xor ebp, ebp
	call main
	mov rdi, rax
	mov eax, 60
	syscall

section .note.GNU-stack noalloc noexec nowrite progbits

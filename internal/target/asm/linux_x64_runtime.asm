global __wisp_emit
global __wisp_error_emit
global __wisp_panic
global __wisp_malloc
global __wisp_free
global __wisp_realloc
extern __wisp_main

global main

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
__wisp_oom_domain: db "runtime.AllocError"
__wisp_oom_message: db "out of memory"
__wisp_oom_file: db "runtime"
__wisp_oom_file_len equ $ - __wisp_oom_file

align 8
__wisp_oom_error:
        dq __wisp_oom_domain
        dq __wisp_oom_message - __wisp_oom_domain
        dq 1
        dq __wisp_oom_message
        dq __wisp_oom_file - __wisp_oom_message
        dq __wisp_oom_file
        dq __wisp_oom_file_len
        dq 0

section .bss
__wisp_heap_list: resq 11
__wisp_heap_cur: resq 1
__wisp_heap_end: resq 1

section .text
__wisp_mmap:
	mov rsi, rdi
	xor edi, edi
	mov edx, 3
	mov r10d, 0x22
	mov r8, -1
	xor r9d, r9d
	mov eax, 9
	syscall
	cmp rax, -4095
	jbe .ok
	xor eax, eax
.ok:
	ret

__wisp_malloc:
	push rbx
	mov rbx, rdi
	cmp rdi, 0
	jl .fail
	
	add rbx, 16
	cmp rbx, 32768
	jg .greater

	cmp rbx, 32
	jbe .needzero
	mov	r8, rbx
	dec r8
	bsr r8, r8
	sub r8, 4
	jmp .bsr_ok
.needzero:
	mov r8, 0
.bsr_ok:

	lea r9, [rel __wisp_heap_list]
	mov rdx, [r9+r8*8]
	test rdx, rdx
	jz .zero
	mov rax, [rdx]
	mov [r9+r8*8], rax
	mov rax, rdx
	xor edx, edx
	pop rbx
	ret
.zero:
	mov r9d, 32
	mov ecx, r8d
	shl r9, cl

	mov rax, [rel __wisp_heap_cur]
	lea r10, [rax+r9]
	cmp r10, [rel __wisp_heap_end]
	jbe .carve
	
	push r8
	push r9
	mov rdi, 1048576
	call __wisp_mmap
	pop r9
	pop r8
	test rax, rax
	jz .fail

	mov [rel __wisp_heap_cur], rax
	add rax, 1048576
	mov [rel __wisp_heap_end], rax

.carve:
	mov rax, [rel __wisp_heap_cur]
	lea r10, [rax+r9]
	mov [rel __wisp_heap_cur], r10
	mov [rax], r8
	add rax, 16
	xor edx, edx
	pop rbx
	ret

.greater:
	add rbx, 4095
	and rbx, -4096
	push rbx
	mov rdi, rbx
	call __wisp_mmap
	pop rbx
	test rax, rax
	jz .fail
	mov [rax], rbx
	add rax, 16
	xor edx, edx
	pop rbx
	ret

.fail:
	pop rbx
	xor eax, eax
	lea rdx, [rel __wisp_oom_error]
	ret

__wisp_free:
	test rdi, rdi
	jz .ok

	mov rax, rdi
	sub rax, 16
	mov r8, [rax]
	cmp r8, 11
	jae .large

	lea r9, [rel __wisp_heap_list]
	mov rdx, [r9 + r8*8]
	mov [rdi], rdx
	mov [r9 + r8*8], rdi
	ret

.large:
	mov rsi, [rax]
	mov rdi, rax
	mov eax, 11
	syscall
.ok:
	ret

__wisp_realloc:
	test rdi, rdi
	jnz .notzero
	mov rdi, rsi
	jmp __wisp_malloc

.notzero:
	mov rax, [rdi-16]
	cmp rax, 11
	jae .lower
	
	mov r9d, 32
	mov ecx, eax
	shl r9, cl
	mov rax, r9
.lower:
	sub rax, 16

	cmp rsi, rax
	ja .need

	mov rax, rdi
	xor edx, edx
	ret

.need:
	push rdi
	push rax
	mov rdi, rsi
	call __wisp_malloc
	mov rdi, rax
	pop rcx
	pop rsi
	test rdx, rdx
	jnz .done

	push rax
	push rsi
	rep movsb
	pop rdi
	call __wisp_free
	pop rax
	xor edx, edx
.done:
	ret

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

main:
	push rbp
	mov rbp, rsp

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
	mov rsi, rdi
	mov rdi, rsp
	call __wisp_main
	leave
	ret

section .note.GNU-stack noalloc noexec nowrite progbits

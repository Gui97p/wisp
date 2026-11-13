default rel

global _start
extern main
extern ExitProcess
extern GetCommandLineW
extern WideCharToMultiByte
extern SetConsoleOutputCP
extern VirtualAlloc

section .text code align=16
_start:
	xor ebp, ebp
	sub rsp, 72

	mov ecx, 65001
	call SetConsoleOutputCP

	call GetCommandLineW
	mov r14, rax

	mov ecx, 65001
	xor edx, edx
	mov r8, r14
	mov r9d, -1
	mov qword [rsp+32], 0
	mov qword [rsp+40], 0
	mov qword [rsp+48], 0
	mov qword [rsp+56], 0
	call WideCharToMultiByte
	test eax, eax
	jz .fail
	mov ebx, eax

	lea rdx, [rbx+rbx*4+32]
	xor ecx, ecx
	mov r8d, 0x3000
	mov r9d, 4
	call VirtualAlloc
	test rax, rax
	jz .fail
	mov r12, rax

	mov rdx, rbx
	shr rdx, 1
	lea r15, [r12+rdx*8+16]

	mov ecx, 65001
	xor edx, edx
	mov r8, r14
	mov r9d, -1
	mov [rsp+32], r15
	mov [rsp+40], rbx
	mov qword [rsp+48], 0
	mov qword [rsp+56], 0
	call WideCharToMultiByte
	test eax, eax
	jz .fail

	mov rsi, r15
	mov rdi, r15
	xor r13d, r13d

	mov [r12+r13*8], rdi
	inc r13
	cmp byte [rsi], '"'
	jne .first_plain
	inc rsi
.first_quoted:
	mov al, [rsi]
	test al, al
	jz .first_done
	inc rsi
	cmp al, '"'
	je .first_done
	mov [rdi], al
	inc rdi
	jmp .first_quoted
.first_plain:
	mov al, [rsi]
	test al, al
	jz .first_done
	cmp al, ' '
	je .first_done
	cmp al, 9
	je .first_done
	mov [rdi], al
	inc rdi
	inc rsi
	jmp .first_plain
.first_done:
	cmp byte [rsi], 0
	je .first_last
	cmp rdi, rsi
	jb .first_store
	inc rsi
.first_store:
	mov byte [rdi], 0
	inc rdi
	jmp .next_arg
.first_last:
	mov byte [rdi], 0
	inc rdi
	jmp .call_main

.next_arg:
	mov al, [rsi]
	cmp al, ' '
	je .skip
	cmp al, 9
	je .skip
	test al, al
	jz .call_main
	mov [r12+r13*8], rdi
	inc r13
	xor r8d, r8d
	jmp .scan
.skip:
	inc rsi
	jmp .next_arg

.scan:
	xor ecx, ecx
.backslashes:
	cmp byte [rsi], 92
	jne .after_backslashes
	inc rsi
	inc ecx
	jmp .backslashes
.after_backslashes:
	mov al, [rsi]
	cmp al, '"'
	jne .plain_char

	mov edx, ecx
	shr edx, 1
.quote_slashes:
	test edx, edx
	jz .quote_slashes_done
	mov byte [rdi], 92
	inc rdi
	dec edx
	jmp .quote_slashes
.quote_slashes_done:
	inc rsi
	test cl, 1
	jnz .literal_quote
	test r8d, r8d
	jnz .inside_quotes
	mov r8d, 1
	jmp .scan
.inside_quotes:
	cmp byte [rsi], '"'
	jne .close_quotes
	inc rsi
	mov byte [rdi], '"'
	inc rdi
	jmp .scan
.close_quotes:
	xor r8d, r8d
	jmp .scan
.literal_quote:
	mov byte [rdi], '"'
	inc rdi
	jmp .scan

.plain_char:
.plain_slashes:
	test ecx, ecx
	jz .plain_slashes_done
	mov byte [rdi], 92
	inc rdi
	dec ecx
	jmp .plain_slashes
.plain_slashes_done:
	test al, al
	jz .token_eof
	test r8d, r8d
	jnz .copy
	cmp al, ' '
	je .token_end
	cmp al, 9
	je .token_end
.copy:
	mov [rdi], al
	inc rdi
	inc rsi
	jmp .scan

.token_end:
	inc rsi
	mov byte [rdi], 0
	inc rdi
	jmp .next_arg
.token_eof:
	mov byte [rdi], 0
	inc rdi

.call_main:
	mov rdi, r13
	mov rsi, r12
	xor edx, edx
	call main
	mov ecx, eax
	call ExitProcess

.fail:
	mov ecx, 1
	call ExitProcess

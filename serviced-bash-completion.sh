#! /bin/bash

PROG=serviced

_cli_bash_autocomplete() {
     local cur prev opts base
     COMPREPLY=()
     cur="${COMP_WORDS[COMP_CWORD]}"
     prev="${COMP_WORDS[COMP_CWORD-1]}"
     opts=$( ${COMP_WORDS[@]:0:COMP_CWORD} --generate-bash-completion )
     if [ $? -eq 0 ]; then
         COMPREPLY=( $(compgen -W "${opts}" -- ${cur}) )
     fi
     return 0
}
  
complete -F _cli_bash_autocomplete $PROG

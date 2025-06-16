_cu_terminal_envs() {
  local expl
  local envs=( ${(f)"$(cu list | awk '{print $1}' )"} )
  compadd -- $envs
}
compdef _cu_terminal_envs 'cu terminal'

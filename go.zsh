# after you source or install _cu …

# 1 – function that spits out env names
_cu_envs() {
  local -a envs
  envs=( ${(f)"$(cu list 2>/dev/null | awk '{print $1}')" } )
  compadd -Q -- $envs            # -Q keeps the slashes in names
}

# 2 – hook it to “cu terminal …”
compdef _cu_envs 'cu:terminal'   # context = command “cu”, subcmd “terminal”


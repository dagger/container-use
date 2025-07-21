package edit

import _ "embed"

//go:embed cmd/edit/main.go
var Src string

//go:embed go.mod
var GoMod string

//go:embed go.sum
var GoSum string

package environment

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNotesAdd(t *testing.T) {
	n := &Notes{}
	n.Add("hello %s", "world")
	n.Add("count %d", 2)
	assert.Contains(t, n.String(), "hello world")
	assert.Contains(t, n.String(), "count 2")
}

func TestNotesAddCommand(t *testing.T) {
	n := &Notes{}
	n.AddCommand("ls /", 0, "file1\nfile2", "")
	assert.Contains(t, n.String(), "$ ls /")
	assert.Contains(t, n.String(), "file1")
	assert.NotContains(t, n.String(), "exit")
	assert.NotContains(t, n.String(), "stderr")
}

func TestNotesAddCommandFailure(t *testing.T) {
	n := &Notes{}
	n.AddCommand("bad", 1, "", "permission denied")
	assert.Contains(t, n.String(), "exit 1")
	assert.Contains(t, n.String(), "stderr: permission denied")
}

func TestNotesClear(t *testing.T) {
	n := &Notes{}
	n.Add("note")
	n.Clear()
	assert.Empty(t, n.String())
}

func TestNotesPop(t *testing.T) {
	n := &Notes{}
	n.Add("note")
	out := n.Pop()
	assert.Equal(t, "note", out)
	assert.Empty(t, n.String())
}

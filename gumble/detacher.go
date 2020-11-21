package gumble // import "github.com/5pm-HDH/gumble/gumble"

// Detacher is an interface that event listeners implement. After the Detach
// method is called, the listener will no longer receive events.
type Detacher interface {
	Detach()
}

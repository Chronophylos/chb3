package state

type Channel struct {
	Name string

	Joined   bool
	Sleeping bool
	Lurking  bool
}

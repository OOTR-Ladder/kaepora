package bot

// Publicly echoed errors
type errPublic string

func (e errPublic) Error() string {
	return string(e)
}

func (e errPublic) Is(v error) bool {
	_, ok := v.(errPublic)
	return ok
}

package target

type Target interface {
	Name() string
	Compile() (string, error)
}

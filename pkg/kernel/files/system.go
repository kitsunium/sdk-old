package files

type System interface {
	Path() string
	Exists() bool
	Size() int64
}

package platformid

type UniquePlatformIDProvider interface {
	Name() string
	ID() (string, error)
}

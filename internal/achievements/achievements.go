package achievements

type Achievement struct {
	Title    string
	Username string
	Value    string

	Kind string // achievement | anti-achievement
	Fact string // сухое описание факта
}

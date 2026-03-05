package skill

//go:generate ormc

type cat struct {
	ID   int64  `db:"pk,autoincrement"`
	Name string `db:"unique"`
}

func (c *cat) TableName() string {
	return "cats"
}

type skillModel struct {
	ID    int64  `db:"pk,autoincrement"`
	CatID int64  `db:"ref=cats"`
	Name  string `db:"unique"`
	Info  string
}

func (s *skillModel) TableName() string {
	return "skills"
}

type paramModel struct {
	ID      int64  `db:"pk,autoincrement"`
	SkillID int64  `db:"ref=skills"`
	Name    string
	Type    string
	Info    string
	Req     bool
}

func (p *paramModel) TableName() string {
	return "params"
}

package envy

type User struct {
	Name string
}

func (u *User) Path(parts ...string) string {
	return Envy.Path(append([]string{"users", u.Name}, parts...)...)
}

func (u *User) Admin() bool {
	if !exists(Envy.Path("config/admins")) {
		writeFile(Envy.Path("config/admins"), u.Name)
	}
	return grepFile(Envy.Path("config/admins"), u.Name)
}

func (u *User) Environ(name string) *Environ {
	return &Environ{
		Name: name,
		User: u,
	}
}

func (u *User) Session(name string) *Session {
	return &Session{
		Name: name,
		User: u,
	}
}

func GetUser(name string) *User {
	u := &User{
		Name: name,
	}
	mkdirAll(u.Path())
	mkdirAll(u.Path("environs"))
	mkdirAll(u.Path("sessions"))
	if !exists(u.Path("home")) {
		mkdirAll(u.Path("home"))
		copy(Envy.DataPath("home", ".bashrc"),
			u.Path("home", ".bashrc"))
	}
	if !exists(u.Path("root")) {
		mkdirAll(u.Path("root"))
		copy(Envy.DataPath("home", ".bashrc"),
			u.Path("root", ".bashrc"))
	}
	return u
}

package errors

var (
	registredErrors = make(map[string]*Error)
)

type Error struct {
	code    uint16
	message string
	tags    map[string]struct{}
}

func (e *Error) Error() string {
	return e.message
}

func (e *Error) Code() uint16 {
	return e.code
}

// HasTag vérifie si l'erreur contient un tag spécifique
func (e *Error) HasTag(tag string) bool {
	_, exists := e.tags[tag]
	return exists
}

// AddTag ajoute un ou plusieurs tags à l'erreur
func (e *Error) AddTag(tags ...string) {
	for _, tag := range tags {
		e.tags[tag] = struct{}{}
	}
}

// RemoveTag supprime un ou plusieurs tags de l'erreur
func (e *Error) RemoveTag(tags ...string) {
	for _, tag := range tags {
		delete(e.tags, tag)
	}
}

// Tags retourne une liste des tags associés à l'erreur
func (e *Error) Tags() []string {
	result := make([]string, 0, len(e.tags))
	for tag := range e.tags {
		result = append(result, tag)
	}
	return result
}

// New crée une nouvelle erreur avec un code, un message et des tags optionnels
func New(code uint16, message string, tags ...string) *Error {
	tagSet := make(map[string]struct{})
	for _, tag := range tags {
		tagSet[tag] = struct{}{}
	}

	err := &Error{
		code:    code,
		message: message,
		tags:    tagSet,
	}

	registredErrors[message] = err

	return err
}

// ListErrors retourne toutes les erreurs enregistrées
func ListErrors() map[string]*Error {
	return registredErrors
}

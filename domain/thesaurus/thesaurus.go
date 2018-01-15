package thesaurus

type Thesaurus interface {
	synonyms(term string) ([]string, error)
}

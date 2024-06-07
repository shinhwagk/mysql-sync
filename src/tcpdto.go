package main

type DTO interface {
	GetGtoString() string
}

// Data Transfer Object
type DTOPause struct {
}

func (d DTOPause) GetGtoString() string {
	return "pause"
}

type DTOResume struct {
}

func (d DTOResume) GetGtoString() string {
	return "resume"
}

type DTOGtidSet struct {
	GtidSet string
}

func (d DTOGtidSet) GetGtoString() string {
	return "gtidSet"
}

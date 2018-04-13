package com

// CDATA CDATA wrapper for xml
type CDATA struct {
	S string `xml:",cdata"`
}

// NewCDATA create a CDATA
func NewCDATA(s string) CDATA {
	return CDATA{S: s}
}

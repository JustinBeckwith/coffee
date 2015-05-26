// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package binding

import (
	"net/http"

	"gopkg.in/bluesuncorp/validator.v5"
)

const (
	MIMEJSON              = "application/json"
	MIMEHTML              = "text/html"
	MIMEXML               = "application/xml"
	MIMEXML2              = "text/xml"
	MIMEPlain             = "text/plain"
	MIMEPOSTForm          = "application/x-www-form-urlencoded"
	MIMEMultipartPOSTForm = "multipart/form-data"
)

type Binding interface {
	Name() string
	Bind(*http.Request, interface{}) error
}

var validate = validator.New("binding", validator.BakedInValidators)

var (
	JSON = jsonBinding{}
	XML  = xmlBinding{}
	Form = formBinding{}
)

func Default(method, contentType string) Binding {
	if method == "GET" {
		return Form
	} else {
		switch contentType {
		case MIMEJSON:
			return JSON
		case MIMEXML, MIMEXML2:
			return XML
		default:
			return Form
		}
	}
}

func ValidateField(f interface{}, tag string) error {
	if err := validate.Field(f, tag); err != nil {
		return error(err)
	}
	return nil
}

func Validate(obj interface{}) error {
	if err := validate.Struct(obj); err != nil {
		return error(err)
	}
	return nil
}

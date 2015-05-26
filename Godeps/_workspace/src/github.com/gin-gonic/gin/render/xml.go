// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package render

import (
	"encoding/xml"
	"net/http"
)

type XML struct {
	Data interface{}
}

const xmlContentType = "application/xml; charset=utf-8"

func (r XML) Write(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", xmlContentType)
	return xml.NewEncoder(w).Encode(r.Data)
}

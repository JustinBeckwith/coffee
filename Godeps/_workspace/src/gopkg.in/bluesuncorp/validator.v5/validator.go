/**
 * Package validator
 *
 * MISC:
 * - anonymous structs - they don't have names so expect the Struct name within StructErrors to be blank
 *
 */

package validator

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"
	"unicode"
)

const (
	tagSeparator    = ","
	orSeparator     = "|"
	noValidationTag = "-"
	tagKeySeparator = "="
	structOnlyTag   = "structonly"
	omitempty       = "omitempty"
	fieldErrMsg     = "Field validation for \"%s\" failed on the \"%s\" tag\n"
	structErrMsg    = "Struct:%s\n"
)

// FieldError contains a single field's validation error along
// with other properties that may be needed for error message creation
type FieldError struct {
	Field string
	Tag   string
	Kind  reflect.Kind
	Type  reflect.Type
	Param string
	Value interface{}
}

// This is intended for use in development + debugging and not intended to be a production error message.
// it also allows FieldError to be used as an Error interface
func (e *FieldError) Error() string {
	return fmt.Sprintf(fieldErrMsg, e.Field, e.Tag)
}

// StructErrors is hierarchical list of field and struct validation errors
// for a non hierarchical representation please see the Flatten method for StructErrors
type StructErrors struct {
	// Name of the Struct
	Struct string
	// Struct Field Errors
	Errors map[string]*FieldError
	// Struct Fields of type struct and their errors
	// key = Field Name of current struct, but internally Struct will be the actual struct name unless anonymous struct, it will be blank
	StructErrors map[string]*StructErrors
}

// This is intended for use in development + debugging and not intended to be a production error message.
// it also allows StructErrors to be used as an Error interface
func (e *StructErrors) Error() string {
	buff := bytes.NewBufferString(fmt.Sprintf(structErrMsg, e.Struct))

	for _, err := range e.Errors {
		buff.WriteString(err.Error())
	}

	for _, err := range e.StructErrors {
		buff.WriteString(err.Error())
	}
	buff.WriteString("\n\n")
	return buff.String()
}

// Flatten flattens the StructErrors hierarchical structure into a flat namespace style field name
// for those that want/need it
func (e *StructErrors) Flatten() map[string]*FieldError {

	if e == nil {
		return nil
	}

	errs := map[string]*FieldError{}

	for _, f := range e.Errors {

		errs[f.Field] = f
	}

	for key, val := range e.StructErrors {

		otherErrs := val.Flatten()

		for _, f2 := range otherErrs {

			f2.Field = fmt.Sprintf("%s.%s", key, f2.Field)
			errs[f2.Field] = f2
		}
	}

	return errs
}

// Func accepts all values needed for file and cross field validation
// top     = top level struct when validating by struct otherwise nil
// current = current level struct when validating by struct otherwise optional comparison value
// f       = field value for validation
// param   = parameter used in validation i.e. gt=0 param would be 0
type Func func(top interface{}, current interface{}, f interface{}, param string) bool

// Validate implements the Validate Struct
// NOTE: Fields within are not thread safe and that is on purpose
// Functions and Tags should all be predifined before use, so subscribe to the philosiphy
// or make it thread safe on your end
type Validate struct {
	// tagName being used.
	tagName string
	// validateFuncs is a map of validation functions and the tag keys
	validationFuncs map[string]Func
}

// New creates a new Validate instance for use.
func New(tagName string, funcs map[string]Func) *Validate {
	return &Validate{
		tagName:         tagName,
		validationFuncs: funcs,
	}
}

// SetTag sets tagName of the Validator to one of your choosing after creation
// perhaps to dodge a tag name conflict in a specific section of code
func (v *Validate) SetTag(tagName string) {
	v.tagName = tagName
}

// AddFunction adds a validation Func to a Validate's map of validators denoted by the key
// NOTE: if the key already exists, it will get replaced.
func (v *Validate) AddFunction(key string, f Func) error {

	if len(key) == 0 {
		return errors.New("Function Key cannot be empty")
	}

	if f == nil {
		return errors.New("Function cannot be empty")
	}

	v.validationFuncs[key] = f

	return nil
}

// Struct validates a struct, even it's nested structs, and returns a struct containing the errors
// NOTE: Nested Arrays, or Maps of structs do not get validated only the Array or Map itself; the reason is that there is no good
// way to represent or report which struct within the array has the error, besides can validate the struct prior to adding it to
// the Array or Map.
func (v *Validate) Struct(s interface{}) *StructErrors {

	return v.structRecursive(s, s, s)
}

// structRecursive validates a struct recursivly and passes the top level and current struct around for use in validator functions and returns a struct containing the errors
func (v *Validate) structRecursive(top interface{}, current interface{}, s interface{}) *StructErrors {

	structValue := reflect.ValueOf(s)
	structType := reflect.TypeOf(s)
	structName := structType.Name()

	if structValue.Kind() == reflect.Ptr && !structValue.IsNil() {
		return v.structRecursive(top, current, structValue.Elem().Interface())
	}

	if structValue.Kind() != reflect.Struct && structValue.Kind() != reflect.Interface {
		panic("interface passed for validation is not a struct")
	}

	validationErrors := &StructErrors{
		Struct:       structName,
		Errors:       map[string]*FieldError{},
		StructErrors: map[string]*StructErrors{},
	}

	var numFields = structValue.NumField()

	for i := 0; i < numFields; i++ {

		valueField := structValue.Field(i)
		typeField := structType.Field(i)

		if valueField.Kind() == reflect.Ptr && !valueField.IsNil() {
			valueField = valueField.Elem()
		}

		tag := typeField.Tag.Get(v.tagName)

		if tag == noValidationTag {
			continue
		}

		// if no validation and not a struct (which may containt fields for validation)
		if tag == "" && ((valueField.Kind() != reflect.Struct && valueField.Kind() != reflect.Interface) || valueField.Type() == reflect.TypeOf(time.Time{})) {
			continue
		}

		switch valueField.Kind() {

		case reflect.Struct, reflect.Interface:

			if !unicode.IsUpper(rune(typeField.Name[0])) {
				continue
			}

			if valueField.Type() == reflect.TypeOf(time.Time{}) {

				if fieldError := v.fieldWithNameAndValue(top, current, valueField.Interface(), typeField.Name, tag); fieldError != nil {
					validationErrors.Errors[fieldError.Field] = fieldError
					// free up memory reference
					fieldError = nil
				}

			} else {

				if strings.Contains(tag, structOnlyTag) {
					continue
				}

				if structErrors := v.structRecursive(top, valueField.Interface(), valueField.Interface()); structErrors != nil {
					validationErrors.StructErrors[typeField.Name] = structErrors
					// free up memory map no longer needed
					structErrors = nil
				}
			}

		default:

			if fieldError := v.fieldWithNameAndValue(top, current, valueField.Interface(), typeField.Name, tag); fieldError != nil {
				validationErrors.Errors[fieldError.Field] = fieldError
				// free up memory reference
				fieldError = nil
			}
		}
	}

	if len(validationErrors.Errors) == 0 && len(validationErrors.StructErrors) == 0 {
		return nil
	}

	return validationErrors
}

// Field allows validation of a single field, still using tag style validation to check multiple errors
func (v *Validate) Field(f interface{}, tag string) *FieldError {

	return v.FieldWithValue(nil, f, tag)
}

// FieldWithValue allows validation of a single field, possibly even against another fields value, still using tag style validation to check multiple errors
func (v *Validate) FieldWithValue(val interface{}, f interface{}, tag string) *FieldError {

	return v.fieldWithNameAndValue(nil, val, f, "", tag)
}

func (v *Validate) fieldWithNameAndValue(val interface{}, current interface{}, f interface{}, name string, tag string) *FieldError {

	// This is a double check if coming from validate.Struct but need to be here in case function is called directly
	if tag == noValidationTag {
		return nil
	}

	if strings.Contains(tag, omitempty) && !hasValue(val, current, f, "") {
		return nil
	}

	valueField := reflect.ValueOf(f)
	fieldKind := valueField.Kind()

	if fieldKind == reflect.Ptr && !valueField.IsNil() {
		return v.fieldWithNameAndValue(val, current, valueField.Elem().Interface(), name, tag)
	}

	fieldType := valueField.Type()

	switch fieldKind {

	case reflect.Struct, reflect.Interface, reflect.Invalid:

		if fieldType != reflect.TypeOf(time.Time{}) {
			panic("Invalid field passed to ValidateFieldWithTag")
		}
	}

	var valErr *FieldError
	var err error
	valTags := strings.Split(tag, tagSeparator)

	for _, valTag := range valTags {

		orVals := strings.Split(valTag, orSeparator)

		if len(orVals) > 1 {

			errTag := ""

			for _, val := range orVals {

				valErr, err = v.fieldWithNameAndSingleTag(val, current, f, name, val)

				if err == nil {
					return nil
				}

				errTag += orSeparator + valErr.Tag

			}

			errTag = strings.TrimLeft(errTag, orSeparator)

			valErr.Tag = errTag
			valErr.Kind = fieldKind

			return valErr
		}

		if valErr, err = v.fieldWithNameAndSingleTag(val, current, f, name, valTag); err != nil {

			valErr.Kind = valueField.Kind()
			valErr.Type = fieldType

			return valErr
		}
	}

	return nil
}

func (v *Validate) fieldWithNameAndSingleTag(val interface{}, current interface{}, f interface{}, name string, valTag string) (*FieldError, error) {

	vals := strings.Split(valTag, tagKeySeparator)
	key := strings.Trim(vals[0], " ")

	if len(key) == 0 {
		panic(fmt.Sprintf("Invalid validation tag on field %s", name))
	}

	valErr := &FieldError{
		Field: name,
		Tag:   key,
		Value: f,
		Param: "",
	}

	// OK to continue because we checked it's existance before getting into this loop
	if key == omitempty {
		return valErr, nil
	}

	valFunc, ok := v.validationFuncs[key]
	if !ok {
		panic(fmt.Sprintf("Undefined validation function on field %s", name))
	}

	param := ""
	if len(vals) > 1 {
		param = strings.Trim(vals[1], " ")
	}

	if err := valFunc(val, current, f, param); !err {
		valErr.Param = param
		return valErr, errors.New(key)
	}

	return valErr, nil
}

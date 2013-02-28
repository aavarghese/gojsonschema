// @author       sigu-399
// @description  An implementation of JSON Schema, draft v4 - Go language
// @created      26-02-2013

package gojsonschema

import (
	"errors"
	"fmt"
	"gojsonreference"
	"reflect"
)

const (
	KEY_SCHEMA      = "$schema"
	KEY_ID          = "$id"
	KEY_REF         = "$ref"
	KEY_TITLE       = "title"
	KEY_DESCRIPTION = "description"
	KEY_TYPE        = "type"
	KEY_ITEMS       = "items"
	KEY_PROPERTIES  = "properties"

	STRING_STRING     = "string"
	STRING_SCHEMA     = "schema"
	STRING_PROPERTIES = "properties"

	ROOT_SCHEMA_PROPERTY = "(root)"
)

func NewJsonSchemaDocument(documentReferenceString string) (*JsonSchemaDocument, error) {

	var err error

	d := JsonSchemaDocument{}
	d.documentReference, err = gojsonreference.NewJsonReference(documentReferenceString)
	d.pool = NewSchemaPool()

	spd, err := d.pool.GetPoolDocument(d.documentReference)
	if err != nil {
		return nil, err
	}

	err = d.parse(spd.Document)
	return &d, err
}

type JsonSchemaDocument struct {
	documentReference gojsonreference.JsonReference
	rootSchema        *JsonSchema
	pool              *SchemaPool
}

func (d *JsonSchemaDocument) parse(document interface{}) error {
	d.rootSchema = &JsonSchema{property: ROOT_SCHEMA_PROPERTY}
	return d.parseSchema(document, d.rootSchema)
}

func (d *JsonSchemaDocument) parseSchema(documentNode interface{}, currentSchema *JsonSchema) error {

	if !isKind(documentNode, reflect.Map) {
		return errors.New(fmt.Sprintf(ERROR_MESSAGE_X_MUST_BE_AN_OBJECT, STRING_SCHEMA))
	}

	m := documentNode.(map[string]interface{})

	if currentSchema == d.rootSchema {
		if !existsMapKey(m, KEY_SCHEMA) {
			return errors.New(fmt.Sprintf(ERROR_MESSAGE_X_IS_REQUIRED, KEY_SCHEMA))
		}
		if !isKind(m[KEY_SCHEMA], reflect.String) {
			return errors.New(fmt.Sprintf(ERROR_MESSAGE_X_MUST_BE_OF_TYPE_Y, KEY_SCHEMA, STRING_STRING))
		}
		schemaRef := m[KEY_SCHEMA].(string)
		schemaReference, err := gojsonreference.NewJsonReference(schemaRef)
		currentSchema.schema = &schemaReference
		if err != nil {
			return err
		}

		currentSchema.ref = &d.documentReference

		if existsMapKey(m, KEY_REF) {
			return errors.New(fmt.Sprintf("No %s is allowed in root schema", KEY_REF))
		}

	}

	// ref
	if existsMapKey(m, KEY_REF) && !isKind(m[KEY_REF], reflect.String) {
		return errors.New(fmt.Sprintf(ERROR_MESSAGE_X_MUST_BE_OF_TYPE_Y, KEY_REF, STRING_STRING))
	}
	if k, ok := m[KEY_REF].(string); ok {
		jsonReference, err := gojsonreference.NewJsonReference(k)
		if err != nil {
			return err
		}

		if jsonReference.HasFullUrl {
			currentSchema.ref = &jsonReference
		} else {
			inheritedReference, err := gojsonreference.Inherits(*currentSchema.ref, jsonReference)
			if err != nil {
				return err
			}
			currentSchema.ref = inheritedReference
		}

		dsp, err := d.pool.GetPoolDocument(*currentSchema.ref)
		if err != nil {
			return err
		}

		jsonPointer := currentSchema.ref.GetPointer()

		httpDocumentNode, err := jsonPointer.Get(dsp.Document)
		if err != nil {
			return err
		}

		if !isKind(httpDocumentNode, reflect.Map) {
			return errors.New(fmt.Sprintf(ERROR_MESSAGE_X_MUST_BE_AN_OBJECT, STRING_SCHEMA))
		}
		m = httpDocumentNode.(map[string]interface{})
	}

	// id
	if existsMapKey(m, KEY_ID) && !isKind(m[KEY_ID], reflect.String) {
		return errors.New(fmt.Sprintf(ERROR_MESSAGE_X_MUST_BE_OF_TYPE_Y, KEY_ID, STRING_STRING))
	}
	if k, ok := m[KEY_ID].(string); ok {
		currentSchema.id = &k
	}

	// title
	if existsMapKey(m, KEY_TITLE) && !isKind(m[KEY_TITLE], reflect.String) {
		return errors.New(fmt.Sprintf(ERROR_MESSAGE_X_MUST_BE_OF_TYPE_Y, KEY_TITLE, STRING_STRING))
	}
	if k, ok := m[KEY_TITLE].(string); ok {
		currentSchema.title = &k
	}

	// description
	if existsMapKey(m, KEY_DESCRIPTION) && !isKind(m[KEY_DESCRIPTION], reflect.String) {
		return errors.New(fmt.Sprintf(ERROR_MESSAGE_X_MUST_BE_OF_TYPE_Y, KEY_DESCRIPTION, STRING_STRING))
	}
	if k, ok := m[KEY_DESCRIPTION].(string); ok {
		currentSchema.description = &k
	}

	// type
	if existsMapKey(m, KEY_TYPE) && !isKind(m[KEY_TYPE], reflect.String) {
		return errors.New(fmt.Sprintf(ERROR_MESSAGE_X_MUST_BE_OF_TYPE_Y, KEY_TYPE, STRING_STRING))
	}
	if k, ok := m[KEY_TYPE].(string); ok {
		if !isStringInSlice(SCHEMA_TYPES, k) {
			return errors.New(fmt.Sprintf("schema %s - %s is invalid", currentSchema.property, KEY_TYPE))
		}
		currentSchema.etype = &k
	} else {
		return errors.New(fmt.Sprintf("schema %s - %s is required", currentSchema.property, KEY_TYPE))
	}

	// properties
	/*	if !existsMapKey(m, KEY_PROPERTIES) {
			return errors.New(fmt.Sprintf(ERROR_MESSAGE_X_IS_REQUIRED, KEY_PROPERTIES))
		}
	*/
	for k := range m {
		if k == KEY_PROPERTIES {
			err := d.parseProperties(m[k], currentSchema)
			if err != nil {
				return err
			}
		}
	}

	// items
	/*	if !existsMapKey(m, KEY_ITEMS) {
			return errors.New(fmt.Sprintf(ERROR_MESSAGE_X_IS_REQUIRED, KEY_ITEMS))
		}
	*/
	for k := range m {
		if k == KEY_ITEMS {
			newSchema := &JsonSchema{parent: currentSchema}
			currentSchema.AddPropertiesChild(newSchema)
			err := d.parseSchema(m[k], newSchema)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (d *JsonSchemaDocument) parseProperties(documentNode interface{}, currentSchema *JsonSchema) error {

	if !isKind(documentNode, reflect.Map) {
		return errors.New(fmt.Sprintf(ERROR_MESSAGE_X_MUST_BE_AN_OBJECT, STRING_PROPERTIES))
	}

	m := documentNode.(map[string]interface{})
	for k := range m {
		schemaProperty := k
		newSchema := &JsonSchema{property: schemaProperty, parent: currentSchema, ref: currentSchema.ref}
		currentSchema.AddPropertiesChild(newSchema)
		err := d.parseSchema(m[k], newSchema)
		if err != nil {
			return err
		}
	}

	return nil
}

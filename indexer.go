package search

import (
	"log"
	"reflect"

	bleve "github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/mapping"
	"github.com/pkg/errors"
)

type Indexer struct {
	indexMapping *mapping.IndexMappingImpl
	indexPath    string
	builder      bleve.Builder

	documemtMappings map[string]*mapping.DocumentMapping
	textAnalizers    map[string]*mapping.FieldMapping
}

type Language interface {
	Language() string
}

func NewIndexer(indexPath string) (*Indexer, error) {
	indexMapping := bleve.NewIndexMapping()

	return &Indexer{
		indexMapping:     indexMapping,
		indexPath:        indexPath,
		documemtMappings: map[string]*mapping.DocumentMapping{},
		textAnalizers:    map[string]*mapping.FieldMapping{},
	}, nil
}

func (i *Indexer) RegisterType(structType interface{}) error {
	docType, err := i.getDocumentType(structType)
	if err != nil {
		return errors.Wrap(err, "failed to get document type")
	}

	if _, ok := i.documemtMappings[docType]; ok {
		return nil
	}

	docMapping := i.getDocumentMapping(structType, "en")

	i.indexMapping.AddDocumentMapping(docType, docMapping)
	i.documemtMappings[docType] = docMapping

	return nil
}

func (i *Indexer) Index(id string, data interface{}) error {
	if i.builder == nil {
		var err error
		i.builder, err = bleve.NewBuilder(i.indexPath, i.indexMapping, nil)
		if err != nil {
			return errors.Wrapf(err, "failed to create %s", i.indexPath)
		}
	}

	return i.builder.Index(id, data)
}

func (i *Indexer) Close() error {
	if i.builder != nil {
		return i.builder.Close()
	}
	return nil
}

func (i *Indexer) getDocumentType(structType interface{}) (string, error) {
	classifier, ok := structType.(mapping.Classifier)
	if !ok {
		return "", errors.New("structType does not implement bleve.Classifier")
	}

	return classifier.Type(), nil
}

func (i *Indexer) getDocumentLanguage(structType interface{}, defaultLang string) string {
	lang, ok := structType.(Language)
	if !ok {
		return defaultLang
	}

	return lang.Language()
}

func (i *Indexer) getDocumentMapping(structType interface{}, defaultLang string) *mapping.DocumentMapping {
	docMapping := mapping.NewDocumentMapping()
	lang := i.getDocumentLanguage(structType, defaultLang)

	reflectType := reflect.TypeOf(structType)
	for f := 0; f < reflectType.NumField(); f++ {
		field := reflectType.Field(f)

		switch field.Type.Kind() {
		case reflect.String:
			log.Printf("field: %s", field.Name)

			intexerTag := field.Tag.Get("indexer")
			if intexerTag == "" {
				continue
			}

			switch intexerTag {
			case "text":
				textFieldMapping := mapping.NewTextFieldMapping()
				textFieldMapping.Analyzer = lang
				docMapping.AddFieldMappingsAt(field.Name, textFieldMapping)

			case "date":
				dateFieldMapping := mapping.NewDateTimeFieldMapping()
				docMapping.AddFieldMappingsAt(field.Name, dateFieldMapping)

			case "no_index":
				noIndexFieldMapping := mapping.NewTextFieldMapping()
				noIndexFieldMapping.Index = false
				docMapping.AddFieldMappingsAt(field.Name, noIndexFieldMapping)

			case "no_store":
				noStoreFieldMapping := mapping.NewTextFieldMapping()
				noStoreFieldMapping.Index = false
				noStoreFieldMapping.Store = false
				docMapping.AddFieldMappingsAt(field.Name, noStoreFieldMapping)
			}

		case reflect.Struct:
			// recursion for nested structs
			fieldValue := reflect.ValueOf(structType).FieldByName(field.Name).Interface()
			docMapping.AddSubDocumentMapping(field.Name, i.getDocumentMapping(fieldValue, lang))
		}
	}

	return docMapping
}

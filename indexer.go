package search

import (
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

	docMapping := i.getDocumentMapping(structType)

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

func (i *Indexer) getDocumentMapping(structType interface{}) *mapping.DocumentMapping {
	docMapping := mapping.NewDocumentMapping()

	reflectType := reflect.TypeOf(structType)
	for i := 0; i < reflectType.NumField(); i++ {
		field := reflectType.Field(i)
		intexerTag := field.Tag.Get("indexer")
		if intexerTag == "" {
			continue
		}

		switch intexerTag {
		case "text":
			textFieldMapping := mapping.NewTextFieldMapping()
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
	}

	return docMapping
}

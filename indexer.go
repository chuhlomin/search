package search

import (
	"reflect"

	bleve "github.com/blevesearch/bleve/v2"
	_ "github.com/blevesearch/bleve/v2/analysis/lang/ar"
	_ "github.com/blevesearch/bleve/v2/analysis/lang/bg"
	_ "github.com/blevesearch/bleve/v2/analysis/lang/ca"
	_ "github.com/blevesearch/bleve/v2/analysis/lang/cjk"
	_ "github.com/blevesearch/bleve/v2/analysis/lang/ckb"
	_ "github.com/blevesearch/bleve/v2/analysis/lang/cs"
	_ "github.com/blevesearch/bleve/v2/analysis/lang/da"
	_ "github.com/blevesearch/bleve/v2/analysis/lang/de"
	_ "github.com/blevesearch/bleve/v2/analysis/lang/el"
	_ "github.com/blevesearch/bleve/v2/analysis/lang/en"
	_ "github.com/blevesearch/bleve/v2/analysis/lang/es"
	_ "github.com/blevesearch/bleve/v2/analysis/lang/eu"
	_ "github.com/blevesearch/bleve/v2/analysis/lang/fa"
	_ "github.com/blevesearch/bleve/v2/analysis/lang/fi"
	_ "github.com/blevesearch/bleve/v2/analysis/lang/fr"
	_ "github.com/blevesearch/bleve/v2/analysis/lang/ga"
	_ "github.com/blevesearch/bleve/v2/analysis/lang/gl"
	_ "github.com/blevesearch/bleve/v2/analysis/lang/hi"
	_ "github.com/blevesearch/bleve/v2/analysis/lang/hr"
	_ "github.com/blevesearch/bleve/v2/analysis/lang/hu"
	_ "github.com/blevesearch/bleve/v2/analysis/lang/hy"
	_ "github.com/blevesearch/bleve/v2/analysis/lang/id"
	_ "github.com/blevesearch/bleve/v2/analysis/lang/in"
	_ "github.com/blevesearch/bleve/v2/analysis/lang/it"
	_ "github.com/blevesearch/bleve/v2/analysis/lang/nl"
	_ "github.com/blevesearch/bleve/v2/analysis/lang/no"
	_ "github.com/blevesearch/bleve/v2/analysis/lang/pt"
	_ "github.com/blevesearch/bleve/v2/analysis/lang/ro"
	_ "github.com/blevesearch/bleve/v2/analysis/lang/ru"
	_ "github.com/blevesearch/bleve/v2/analysis/lang/sv"
	_ "github.com/blevesearch/bleve/v2/analysis/lang/tr"
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

func (i *Indexer) RegisterType(structType interface{}, lang string) error {
	docType := i.getDocumentType(structType)

	if _, ok := i.documemtMappings[docType]; ok {
		return nil
	}

	docMapping := i.getDocumentMapping(structType, lang)

	i.indexMapping.AddDocumentMapping(docType, docMapping)
	i.documemtMappings[docType] = docMapping

	return nil
}

func (i *Indexer) Index(id string, data interface{}) error {
	if i.builder == nil {
		var err error
		config := map[string]interface{}{
			"buildPathPrefix": i.indexPath + "_temp",
		}
		i.builder, err = bleve.NewBuilder(i.indexPath, i.indexMapping, config)
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

func (i *Indexer) getDocumentType(structType interface{}) string {
	classifier, ok := structType.(mapping.Classifier)
	if !ok {
		reflectType := reflect.TypeOf(structType)
		return reflectType.Name()
	}

	return classifier.Type()
}

func (i *Indexer) getDocumentLanguage(structType interface{}, defaultLang string) string {
	lang, ok := structType.(Language)
	if !ok {
		return defaultLang
	}

	result := lang.Language()
	if result == "" {
		return defaultLang
	}
	return result
}

func (i *Indexer) getDocumentMapping(structType interface{}, defaultLang string) *mapping.DocumentMapping {
	docMapping := mapping.NewDocumentMapping()
	lang := i.getDocumentLanguage(structType, defaultLang)

	reflectType := reflect.TypeOf(structType)
	for f := 0; f < reflectType.NumField(); f++ {
		field := reflectType.Field(f)

		switch field.Type.Kind() {
		case reflect.String:
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

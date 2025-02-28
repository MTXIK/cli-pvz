package service

import (
	"errors"

	"gitlab.ozon.dev/gojhw1/pkg/model"
)

var (
	ErrUnknownPackageType = errors.New("неизвестный тип упаковки")
)

type packagerFactory interface {
	createPackager(baseType *model.PackageType, wrappers *model.WrapperType) (packager, error)
}

type defaultPackagerFactory struct{}

func newPackagerFactory() packagerFactory {
	return &defaultPackagerFactory{}
}

func (f *defaultPackagerFactory) createPackager(baseType *model.PackageType, wrapper *model.WrapperType) (packager, error) {
	if baseType == nil {
		return nil, ErrUnknownPackageType
	}

	var basePackager packager
	switch *baseType {
	case model.PackageBag:
		basePackager = newBagPackager()
	case model.PackageBox:
		basePackager = newBoxPackager()
	case model.PackageFilm:
		basePackager = newFilmPackager()
	default:
		return nil, ErrUnknownPackageType
	}

	if wrapper != nil {
		decorated, err := newWrapperDecorator(basePackager, *wrapper)
		if err != nil {
			return nil, err
		}
		return decorated, nil
	}

	return basePackager, nil
}

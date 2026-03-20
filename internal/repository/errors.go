package repository

import "errors"

var ErrNotFound = errors.New("resource not found")
var ErrInvalidID = errors.New("invalid id")

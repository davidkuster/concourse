// Code generated by mockery v2.8.0. DO NOT EDIT.

package cmocks

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
)

// Drainable is an autogenerated mock type for the Drainable type
type Drainable struct {
	mock.Mock
}

// Drain provides a mock function with given fields: _a0
func (_m *Drainable) Drain(_a0 context.Context) {
	_m.Called(_a0)
}
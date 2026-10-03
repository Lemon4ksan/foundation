package mock_test

import (
	"fmt"
	"testing"

	"github.com/lemon4ksan/foundation/testing/mock"
)

type mockTestService struct {
	ctrl *mock.Controller
}

func (m *mockTestService) DoWork(msg string, count int) (int, error) {
	ret := m.ctrl.Call(m, "DoWork", msg, count)
	
	var r0 int
	if ret[0] != nil {
		r0 = ret[0].(int)
	}
	
	var r1 error
	if ret[1] != nil {
		r1 = ret[1].(error)
	}
	
	return r0, r1
}

func (m *mockTestService) EXPECT() *mockTestServiceRecorder {
	return &mockTestServiceRecorder{mock: m}
}

type mockTestServiceRecorder struct {
	mock *mockTestService
}

func (r *mockTestServiceRecorder) DoWork(msg, count any) *mock.Call {
	return r.mock.ctrl.RecordCall(r.mock, "DoWork", msg, count)
}

func Example() {
	var t *testing.T
	ctrl := mock.NewController(t)
	defer ctrl.Finish()

	m := &mockTestService{ctrl: ctrl}
	
	m.EXPECT().DoWork(mock.Any(), 10).Return(42, nil)

	res, err := m.DoWork("hello", 10)
	fmt.Printf("Result: %d, Error: %v\n", res, err)
}

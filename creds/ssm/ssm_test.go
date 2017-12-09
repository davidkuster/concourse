package ssm_test

import (
	"errors"
	"strconv"
	"text/template"

	"code.cloudfoundry.org/lager"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/aws/aws-sdk-go/service/ssm/ssmiface"
	varTemplate "github.com/cloudfoundry/bosh-cli/director/template"
	. "github.com/concourse/atc/creds/ssm"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
)

type mockPathResultPage struct {
	params []string
	err    error
}

func (page mockPathResultPage) ToGetParametersByPathOutput() (*ssm.GetParametersByPathOutput, error) {
	if page.err != nil {
		return nil, page.err
	}
	params := make([]*ssm.Parameter, len(page.params))
	for i, p := range page.params {
		params[i] = &ssm.Parameter{
			Name:  aws.String(p),
			Value: aws.String(p + ":value"),
		}
	}
	return &ssm.GetParametersByPathOutput{Parameters: params}, nil
}

type MockSsmService struct {
	ssmiface.SSMAPI

	stubGetParameter             func(name string) (string, error)
	stubGetParametersByPathPages func(path string) []mockPathResultPage
}

func (mock *MockSsmService) GetParameter(input *ssm.GetParameterInput) (*ssm.GetParameterOutput, error) {
	if mock.stubGetParameter == nil {
		return nil, errors.New("stubGetParameter is not defined")
	}
	Expect(input).ToNot(BeNil())
	Expect(input.Name).ToNot(BeNil())
	Expect(input.WithDecryption).To(PointTo(Equal(true)))
	value, err := mock.stubGetParameter(*input.Name)
	if err != nil {
		return nil, err
	}
	return &ssm.GetParameterOutput{Parameter: &ssm.Parameter{Value: &value}}, nil
}

func (mock *MockSsmService) GetParametersByPathPages(input *ssm.GetParametersByPathInput, fn func(*ssm.GetParametersByPathOutput, bool) bool) error {
	if mock.stubGetParametersByPathPages == nil {
		return errors.New("stubGetParametersByPathPages is not defined")
	}
	Expect(input).NotTo(BeNil())
	Expect(input.Path).NotTo(BeNil())
	Expect(input.Recursive).To(PointTo(Equal(true)))
	Expect(input.WithDecryption).To(PointTo(Equal(false)))
	Expect(input.MaxResults).To(PointTo(BeEquivalentTo(10)))
	allPages := mock.stubGetParametersByPathPages(*input.Path)
	if len(allPages) == 0 {
		return errors.New("no pages are returned by the stub")
	}
	for n, page := range allPages {
		params, err := page.ToGetParametersByPathOutput()
		if err != nil {
			return err
		}
		params.NextToken = aws.String(strconv.Itoa(n + 1))
		lastPage := (n == len(allPages)-1)
		if !fn(params, lastPage) {
			break
		}
	}
	return nil
}

var _ = Describe("Ssm", func() {
	var ssmAccess *Ssm
	var varDef varTemplate.VariableDefinition
	var mockService MockSsmService

	JustBeforeEach(func() {
		secretTemplate, err := template.New("test").Parse(DefaultSecretTemplate)
		Expect(secretTemplate).NotTo(BeNil())
		Expect(err).To(BeNil())
		fallbackTemplate, err := template.New("test").Parse(DefaultFallbackTemplate)
		Expect(fallbackTemplate).NotTo(BeNil())
		Expect(err).To(BeNil())
		ssmAccess = NewSsm(lager.NewLogger("ssm_test"), &mockService, "alpha", "bogus", secretTemplate, fallbackTemplate)
		Expect(ssmAccess).NotTo(BeNil())
		varDef = varTemplate.VariableDefinition{Name: "cheery"}
		mockService.stubGetParameter = func(input string) (string, error) {
			Expect(input).To(Equal("/concourse/alpha/bogus/cheery"))
			return "ssm decrypted value", nil
		}
		mockService.stubGetParametersByPathPages = func(path string) []mockPathResultPage {
			return []mockPathResultPage{
				{
					params: []string{"/concourse/alpha/bogus/cheery"},
					err:    nil,
				},
			}
		}
	})

	Describe("Get()", func() {
		It("should get parameter if exists", func() {
			value, found, err := ssmAccess.Get(varDef)
			Expect(value).To(BeEquivalentTo("ssm decrypted value"))
			Expect(found).To(BeTrue())
			Expect(err).To(BeNil())
		})

		It("should get fallback parameter if exists", func() {
			mockService.stubGetParameter = func(input string) (string, error) {
				if input != "/concourse/alpha/cheery" {
					return "", errors.New("parameter not found")
				}
				return "fallback decrypted value", nil
			}
			value, found, err := ssmAccess.Get(varDef)
			Expect(value).To(BeEquivalentTo("fallback decrypted value"))
			Expect(found).To(BeTrue())
			Expect(err).To(BeNil())
		})

		It("should return not found on error", func() {
			mockService.stubGetParameter = nil
			value, found, err := ssmAccess.Get(varDef)
			Expect(value).To(BeNil())
			Expect(found).To(BeFalse())
			Expect(err).NotTo(BeNil())
		})

		It("should work even if fallback template is nil", func() {
			svcErr := errors.New("parameter not found")
			ssmAccess.FallbackTemplate = nil
			mockService.stubGetParameter = func(input string) (string, error) {
				if input != "/concourse/alpha/cheery" {
					return "", svcErr
				}
				return "fallback decrypted value", nil
			}
			value, found, err := ssmAccess.Get(varDef)
			Expect(value).To(BeNil())
			Expect(found).To(BeFalse())
			Expect(err).To(Equal(svcErr))
		})
	})

	Describe("List()", func() {
		It("should list all parameters once", func() {
			mockService.stubGetParametersByPathPages = func(path string) []mockPathResultPage {
				return []mockPathResultPage{
					{
						params: []string{"/concourse/alpha/bogus/cheery"},
						err:    nil,
					},
					{
						params: []string{"/concourse/alpha/bogus/cheery", "/concourse/alpha/bogus/cocco"},
						err:    nil,
					},
				}
			}
			vars, err := ssmAccess.List()
			Expect(err).To(BeNil())
			Expect(vars).To(Equal([]varTemplate.VariableDefinition{
				{Name: "/concourse/alpha/bogus/cheery"},
				{Name: "/concourse/alpha/bogus/cocco"},
			}))
		})
	})
})

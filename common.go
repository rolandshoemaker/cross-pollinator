package cross

import (
	"crypto/x509/pkix"
	"fmt"
	"strings"

	ctPKIX "github.com/google/certificate-transparency/go/x509/pkix"
)

func subjectToString(subject pkix.Name) string {
	out := []string{}
	if subject.CommonName != "" {
		out = append(out, fmt.Sprintf("CN=%s", subject.CommonName))
	}
	if len(subject.Organization) != 0 {
		out = append(out, fmt.Sprintf("O=[%s]", strings.Join(subject.Organization, ", ")))
	}
	if len(subject.OrganizationalUnit) != 0 {
		out = append(out, fmt.Sprintf("OU=[%s]", strings.Join(subject.OrganizationalUnit, ", ")))
	}
	if len(subject.Locality) != 0 {
		out = append(out, fmt.Sprintf("L=[%s]", strings.Join(subject.Locality, ", ")))
	}
	if len(subject.Province) != 0 {
		out = append(out, fmt.Sprintf("ST=[%s]", strings.Join(subject.Province, ", ")))
	}
	if len(subject.Country) != 0 {
		out = append(out, fmt.Sprintf("C=[%s]", strings.Join(subject.Country, ", ")))
	}
	if len(out) == 0 {
		return "???"
	}
	return strings.Join(out, "; ")
}

func ctSubjectToString(subject ctPKIX.Name) string {
	out := []string{}
	if subject.CommonName != "" {
		out = append(out, fmt.Sprintf("CN=%s", subject.CommonName))
	}
	if len(subject.Organization) != 0 {
		out = append(out, fmt.Sprintf("O=[%s]", strings.Join(subject.Organization, ", ")))
	}
	if len(subject.OrganizationalUnit) != 0 {
		out = append(out, fmt.Sprintf("OU=[%s]", strings.Join(subject.OrganizationalUnit, ", ")))
	}
	if len(subject.Locality) != 0 {
		out = append(out, fmt.Sprintf("L=[%s]", strings.Join(subject.Locality, ", ")))
	}
	if len(subject.Province) != 0 {
		out = append(out, fmt.Sprintf("ST=[%s]", strings.Join(subject.Province, ", ")))
	}
	if len(subject.Country) != 0 {
		out = append(out, fmt.Sprintf("C=[%s]", strings.Join(subject.Country, ", ")))
	}
	if len(out) == 0 {
		return "???"
	}
	return strings.Join(out, "; ")
}

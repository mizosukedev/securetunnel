package proxy

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/suite"
)

type ServiceConfigTest struct {
	suite.Suite
}

func TestServiceConfig(t *testing.T) {
	suite.Run(t, new(ServiceConfigTest))
}

func (suite *ServiceConfigTest) TestParseServiceConfig() {
	type args struct {
		serviceStr string
	}
	tests := []struct {
		name    string
		args    args
		want    []ServiceConfig
		wantErr bool
	}{
		// Add test cases.
		{"normal1", args{"22"}, []ServiceConfig{{"", "tcp", "localhost:22"}}, false},
		{"normal2", args{"ssh=22"}, []ServiceConfig{{"ssh", "tcp", "localhost:22"}}, false},
		{"normal3", args{"rdp=localhost:3389"}, []ServiceConfig{{"rdp", "tcp", "localhost:3389"}}, false},
		{"normal4", args{"http=unix:///tmp/http.sock"}, []ServiceConfig{{"http", "unix", "/tmp/http.sock"}}, false},
		{"multiple services", args{"ssh=22, rdp=3389"}, []ServiceConfig{{"ssh", "tcp", "localhost:22"}, {"rdp", "tcp", "localhost:3389"}}, false},
		{"invalid format1", args{"ssh=22=a"}, []ServiceConfig{}, true},
		{"invalid format2", args{"ssh=unix:///tmp/ssh.sock://ssh.sock"}, []ServiceConfig{}, true},
		{"invalid format3", args{"ssh=localhost:22:10022"}, []ServiceConfig{}, true},
		{"port out of range", args{"65536"}, []ServiceConfig{}, true},
	}
	for _, tt := range tests {
		suite.Run(tt.name, func() {
			got, err := ParseServiceConfig(tt.args.serviceStr)
			if (err != nil) != tt.wantErr {
				suite.Fail("ParseServiceConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				suite.Fail("ParseServiceConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}

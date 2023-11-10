package storage

import (
	"os"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/defaults"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/pkg/errors"
	"github.com/wal-g/tracelog"
)

type SessionPool struct {
}

func configWithSettings(s *session.Session, bucket string, settings map[string]string) (*aws.Config, error) {
	// DefaultRetryer implements basic retry logic using exponential backoff for
	// most services. If you want to implement custom retry logic, you can implement the
	// request.Retryer interface.
	maxRetriesCount := MaxRetriesDefault
	if maxRetriesRaw, ok := settings[MaxRetriesSetting]; ok {
		maxRetriesInt, err := strconv.Atoi(maxRetriesRaw)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse %s", MaxRetriesSetting)
		}

		maxRetriesCount = maxRetriesInt
	}
	config := s.Config
	config = request.WithRetryer(config, NewConnResetRetryer(client.DefaultRetryer{NumMaxRetries: maxRetriesCount}))

	accessKeyID := getFirstSettingOf(settings, []string{AccessKeyIDSetting, AccessKeySetting})
	secretAccessKey := getFirstSettingOf(settings, []string{SecretAccessKeySetting, SecretKeySetting})
	sessionToken := settings[SessionTokenSetting]

	roleArn := settings[RoleARN]
	sessionName := settings[SessionName]
	if roleArn != "" {
		stsSession := sts.New(s)
		assumedRole, err := stsSession.AssumeRole(&sts.AssumeRoleInput{
			RoleArn:         aws.String(roleArn),
			RoleSessionName: aws.String(sessionName),
		})
		if err != nil {
			return nil, err
		}
		accessKeyID = *assumedRole.Credentials.AccessKeyId
		secretAccessKey = *assumedRole.Credentials.SecretAccessKey
		sessionToken = *assumedRole.Credentials.SessionToken
	}

	if accessKeyID != "" && secretAccessKey != "" {
		provider := &credentials.StaticProvider{Value: credentials.Value{
			AccessKeyID:     accessKeyID,
			SecretAccessKey: secretAccessKey,
			SessionToken:    sessionToken,
		}}
		providers := make([]credentials.Provider, 0)
		providers = append(providers, provider)
		providers = append(providers, defaults.CredProviders(config, defaults.Handlers())...)
		newCredentials := credentials.NewCredentials(&credentials.ChainProvider{
			VerboseErrors: aws.BoolValue(config.CredentialsChainVerboseErrors),
			Providers:     providers,
		})

		config = config.WithCredentials(newCredentials)
	}

	if logLevel, ok := settings[LogLevel]; ok {
		config = config.WithLogLevel(func(s string) aws.LogLevelType {
			switch s {
			case "DEVEL":
				return aws.LogDebug
			default:
				return aws.LogOff
			}
		}(logLevel))
	}

	if endpoint, ok := settings[EndpointSetting]; ok {
		config = config.WithEndpoint(endpoint)
	}

	if s3ForcePathStyleStr, ok := settings[ForcePathStyleSetting]; ok {
		s3ForcePathStyle, err := strconv.ParseBool(s3ForcePathStyleStr)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse %s", ForcePathStyleSetting)
		}
		config.S3ForcePathStyle = aws.Bool(s3ForcePathStyle)
	}

	region, err := getAWSRegion(bucket, config, settings)
	if err != nil {
		return nil, err
	}
	config = config.WithRegion(region)

	return config, nil
}

// TODO : unit tests
func createSession(bucket string, settings map[string]string) (*session.Session, error) {
	s, err := session.NewSession()
	if err != nil {
		return nil, err
	}

	c, err := configWithSettings(s, bucket, settings)
	if err != nil {
		return nil, err
	}
	s.Config = c

	filePath := settings[s3CertFile]
	if filePath != "" {
		if file, err := os.Open(filePath); err == nil {
			defer file.Close()
			s, err := session.NewSessionWithOptions(session.Options{Config: *s.Config, CustomCABundle: file})
			return s, err
		}
		return nil, err
	}

	if endpointSource, ok := settings[EndpointSourceSetting]; ok {
		s.Handlers.Validate.PushBack(func(request *request.Request) {
			src := setupReqProxy(endpointSource, getEndpointPort(settings))
			if src != nil {
				tracelog.DebugLogger.Printf("using endpoint %s", *src)
				host := strings.TrimPrefix(*s.Config.Endpoint, "https://")
				request.HTTPRequest.Host = host
				request.HTTPRequest.Header.Add("Host", host)
				request.HTTPRequest.URL.Host = *src
				request.HTTPRequest.URL.Scheme = HTTP
			} else {
				tracelog.DebugLogger.Printf("using endpoint %s", *s.Config.Endpoint)
			}
		})
	}

	if encodedHeaders, ok := settings[RequestAdditionalHeaders]; ok {
		headers, err := getHeaders(encodedHeaders)
		if err != nil {
			return nil, err
		}

		s.Handlers.Validate.PushBack(func(request *request.Request) {
			for k, v := range headers {
				request.HTTPRequest.Header.Add(k, v)
			}
		})
	}

	return s, err
}

func NewStorage(bucket, name string) {
	sess, err := createSession(bucket, settings)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create new session")
	}
	client := s3.New(sess)
}

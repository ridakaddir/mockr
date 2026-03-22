package grpc

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/ridakaddir/mockr/internal/config"
	"github.com/ridakaddir/mockr/internal/logger"
	"github.com/ridakaddir/mockr/internal/persist"
	"google.golang.org/grpc/codes"
)

// applyGRPCPersist handles persist operations for a matched gRPC route case.
// It mutates the stub file on disk (append / replace / delete), logs the result,
// and returns (grpcCode, handled). The caller always sends an empty proto message
// on success — unary gRPC requires exactly one response frame.
func (h *handler) applyGRPCPersist(
	c config.Case,
	reqMap map[string]interface{},
	start time.Time,
	fullMethod string,
) (code codes.Code, handled bool) {
	configDir := h.loader.ConfigDir()
	filePath := resolveGRPCFilePath(c.File, reqMap, configDir)

	switch strings.ToLower(c.Merge) {

	case "append":
		if err := persist.Append(filePath, c.ArrayKey, reqMap); err != nil {
			logger.Error("grpc persist append", "file", filePath, "err", err)
			logger.LogGRPC(fullMethod, codes.Internal, time.Since(start), logger.SourceStub)
			return codes.Internal, true
		}
		logger.LogGRPC(fullMethod, codes.OK, time.Since(start), logger.SourceStub)
		return codes.OK, true

	case "replace":
		keyVal := resolveGRPCKeyValue(c.Key, reqMap)
		if keyVal == "" {
			logger.Warn("grpc persist replace: key not found in request", "key", c.Key)
			logger.LogGRPC(fullMethod, codes.InvalidArgument, time.Since(start), logger.SourceStub)
			return codes.InvalidArgument, true
		}
		if _, err := persist.Replace(filePath, c.ArrayKey, c.Key, keyVal, reqMap); err != nil {
			if persist.IsNotFound(err) {
				logger.LogGRPC(fullMethod, codes.NotFound, time.Since(start), logger.SourceStub)
				return codes.NotFound, true
			}
			logger.Error("grpc persist replace", "file", filePath, "err", err)
			logger.LogGRPC(fullMethod, codes.Internal, time.Since(start), logger.SourceStub)
			return codes.Internal, true
		}
		logger.LogGRPC(fullMethod, codes.OK, time.Since(start), logger.SourceStub)
		return codes.OK, true

	case "delete":
		keyVal := resolveGRPCKeyValue(c.Key, reqMap)
		if keyVal == "" {
			logger.Warn("grpc persist delete: key not found in request", "key", c.Key)
			logger.LogGRPC(fullMethod, codes.InvalidArgument, time.Since(start), logger.SourceStub)
			return codes.InvalidArgument, true
		}
		if err := persist.Delete(filePath, c.ArrayKey, c.Key, keyVal); err != nil {
			if persist.IsNotFound(err) {
				logger.LogGRPC(fullMethod, codes.NotFound, time.Since(start), logger.SourceStub)
				return codes.NotFound, true
			}
			logger.Error("grpc persist delete", "file", filePath, "err", err)
			logger.LogGRPC(fullMethod, codes.Internal, time.Since(start), logger.SourceStub)
			return codes.Internal, true
		}
		logger.LogGRPC(fullMethod, codes.OK, time.Since(start), logger.SourceStub)
		return codes.OK, true

	default:
		logger.Warn("grpc persist: unknown merge strategy", "merge", c.Merge)
		logger.LogGRPC(fullMethod, codes.Internal, time.Since(start), logger.SourceStub)
		return codes.Internal, true
	}
}

// resolveGRPCKeyValue extracts the key value from the decoded request map.
// Tries the field name as-is first, then the camelCase conversion so that
// both "item_id" and "itemId" work.
func resolveGRPCKeyValue(key string, reqMap map[string]interface{}) string {
	if v, ok := reqMap[key]; ok {
		return fmt.Sprintf("%v", v)
	}
	if camel := snakeToCamel(key); camel != key {
		if v, ok := reqMap[camel]; ok {
			return fmt.Sprintf("%v", v)
		}
	}
	return ""
}

// resolveGRPCFilePath resolves {body.field} placeholders in the file path and
// makes it absolute relative to configDir.
var grpcPlaceholderRe = regexp.MustCompile(`\{body\.([^}]+)\}`)

func resolveGRPCFilePath(filePath string, reqMap map[string]interface{}, configDir string) string {
	if grpcPlaceholderRe.MatchString(filePath) {
		filePath = grpcPlaceholderRe.ReplaceAllStringFunc(filePath, func(match string) string {
			sub := grpcPlaceholderRe.FindStringSubmatch(match)
			if len(sub) != 2 {
				return match
			}
			field := sub[1]
			for _, key := range []string{field, snakeToCamel(field)} {
				if v, ok := reqMap[key]; ok {
					s := fmt.Sprintf("%v", v)
					return regexp.MustCompile(`[^a-zA-Z0-9_\-.]`).ReplaceAllString(s, "_")
				}
			}
			return match
		})
	}
	if configDir != "" && !filepath.IsAbs(filePath) {
		filePath = filepath.Join(configDir, filePath)
	}
	return filePath
}

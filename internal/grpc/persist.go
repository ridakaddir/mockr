package grpc

import (
	"fmt"
	"os"
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

	// Note: Persist operations return the updated/created data but we ignore it
	// because the gRPC handler sends an empty response for persist operations.
	// This is by design - persist-enabled RPCs should use empty response messages.
	switch strings.ToLower(c.Merge) {

	case "update":
		if _, err := persist.Update(filePath, reqMap); err != nil {
			if persist.IsNotFound(err) {
				logger.LogGRPC(fullMethod, codes.NotFound, time.Since(start), logger.SourceStub)
				return codes.NotFound, true
			}
			if persist.IsConfigError(err) {
				logger.Error("grpc persist update config error", "file", filePath, "err", err)
				logger.LogGRPC(fullMethod, codes.InvalidArgument, time.Since(start), logger.SourceStub)
				return codes.InvalidArgument, true
			}
			logger.Error("grpc persist update", "file", filePath, "err", err)
			logger.LogGRPC(fullMethod, codes.Internal, time.Since(start), logger.SourceStub)
			return codes.Internal, true
		}
		logger.LogGRPC(fullMethod, codes.OK, time.Since(start), logger.SourceStub)
		return codes.OK, true

	case "append":
		if !isGRPCDirectoryPath(filePath, c.File) {
			logger.Error("grpc persist append", "file", filePath, "err", "append requires directory path")
			logger.LogGRPC(fullMethod, codes.InvalidArgument, time.Since(start), logger.SourceStub)
			return codes.InvalidArgument, true
		}
		if _, err := persist.AppendToDir(filePath, c.Key, reqMap); err != nil {
			logger.Error("grpc persist append to dir", "dir", filePath, "err", err)
			logger.LogGRPC(fullMethod, codes.Internal, time.Since(start), logger.SourceStub)
			return codes.Internal, true
		}
		logger.LogGRPC(fullMethod, codes.OK, time.Since(start), logger.SourceStub)
		return codes.OK, true

	case "delete":
		if err := persist.DeleteFile(filePath); err != nil {
			if persist.IsNotFound(err) {
				logger.LogGRPC(fullMethod, codes.NotFound, time.Since(start), logger.SourceStub)
				return codes.NotFound, true
			}
			logger.Error("grpc persist delete file", "file", filePath, "err", err)
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

// isGRPCDirectoryPath determines if a file path should be treated as a directory.
func isGRPCDirectoryPath(resolvedPath, originalConfigFile string) bool {
	info, err := os.Stat(resolvedPath)
	if err == nil && info.IsDir() {
		return true
	}
	// If path doesn't exist but original config indicated directory intent
	if os.IsNotExist(err) && strings.HasSuffix(originalConfigFile, "/") {
		return true
	}
	return false
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

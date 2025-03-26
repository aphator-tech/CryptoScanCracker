package utils

import (
        "bufio"
        "os"
        "strconv"
        "strings"
        "sync"
)

var (
        envCache     = make(map[string]string)
        envCacheMux  sync.RWMutex
        envCacheInit bool
        runtimeCache = make(map[string]string)
        runtimeMux   sync.RWMutex
)

// ReadEnv reads a value from env.txt
func ReadEnv(key string) (string, bool) {
        // Initialize cache if not already done
        if !envCacheInit {
                loadEnvCache()
        }

        envCacheMux.RLock()
        defer envCacheMux.RUnlock()
        
        val, ok := envCache[key]
        return val, ok
}

// ReadEnvBool reads a boolean value from env.txt
func ReadEnvBool(key string) (bool, bool) {
        val, ok := ReadEnv(key)
        if !ok {
                return false, false
        }
        
        // Convert string to boolean
        val = strings.TrimSpace(strings.ToLower(val))
        return val == "true" || val == "1" || val == "yes" || val == "y", true
}

// ReadEnvInt reads an integer value from env.txt
func ReadEnvInt(key string) (int, bool) {
        val, ok := ReadEnv(key)
        if !ok {
                return 0, false
        }
        
        // Convert string to integer
        i, err := strconv.Atoi(strings.TrimSpace(val))
        if err != nil {
                return 0, false
        }
        
        return i, true
}

// ReadEnvFloat reads a float value from env.txt
func ReadEnvFloat(key string) (float64, bool) {
        val, ok := ReadEnv(key)
        if !ok {
                return 0, false
        }
        
        // Convert string to float
        f, err := strconv.ParseFloat(strings.TrimSpace(val), 64)
        if err != nil {
                return 0, false
        }
        
        return f, true
}

// loadEnvCache loads the contents of env.txt into memory
func loadEnvCache() {
        envCacheMux.Lock()
        defer envCacheMux.Unlock()
        
        // Clear existing cache
        for k := range envCache {
                delete(envCache, k)
        }
        
        // Open the env file
        file, err := os.Open("env.txt")
        if err != nil {
                // If file doesn't exist, just return
                return
        }
        defer file.Close()
        
        // Read the file line by line
        scanner := bufio.NewScanner(file)
        for scanner.Scan() {
                line := scanner.Text()
                
                // Skip empty lines and comments
                if line == "" || strings.HasPrefix(line, "#") {
                        continue
                }
                
                // Split the line into key and value
                parts := strings.SplitN(line, "=", 2)
                if len(parts) != 2 {
                        continue
                }
                
                key := strings.TrimSpace(parts[0])
                value := strings.TrimSpace(parts[1])
                
                // Store in cache
                envCache[key] = value
        }
        
        envCacheInit = true
}

// SetRuntimeValue sets a runtime value
func SetRuntimeValue(key, value string) {
        runtimeMux.Lock()
        defer runtimeMux.Unlock()
        runtimeCache[key] = value
}

// GetRuntimeValue gets a runtime value
func GetRuntimeValue(key string) (string, bool) {
        runtimeMux.RLock()
        defer runtimeMux.RUnlock()
        val, ok := runtimeCache[key]
        return val, ok
}

// GetRuntimeBool gets a runtime boolean value
func GetRuntimeBool(key string) (bool, bool) {
        val, ok := GetRuntimeValue(key)
        if !ok {
                return false, false
        }
        
        val = strings.TrimSpace(strings.ToLower(val))
        return val == "true" || val == "1" || val == "yes" || val == "y", true
}
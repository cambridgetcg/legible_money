// Package main implements a simple HTTP faucet for the LGM devnet/testnet.
//
// It wraps the legbled CLI to sign and broadcast bank send transactions.
// The legbled binary must be available in PATH.
//
// Environment variables:
//
//	FAUCET_KEY_NAME   Key name in keyring (default: faucet)
//	FAUCET_AMOUNT     Amount per request (default: 10000000ulgm = 10 LGM)
//	CHAIN_ID          Chain ID (default: lgm-devnet-1)
//	NODE_URL          Node RPC endpoint (default: http://localhost:26657)
//	KEYRING_DIR       Keyring home directory (default: /root/.legbled)
//	LISTEN_ADDR       HTTP listen address (default: :8088)
//	FAUCET_MNEMONIC   If set, recovers the faucet key at startup
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"
)

var (
	faucetKeyName = envOr("FAUCET_KEY_NAME", "faucet")
	faucetAmount  = envOr("FAUCET_AMOUNT", "10000000ulgm")
	chainID       = envOr("CHAIN_ID", "lgm-devnet-1")
	nodeURL       = envOr("NODE_URL", "http://localhost:26657")
	keyringDir    = envOr("KEYRING_DIR", "/root/.legbled")
	listenAddr    = envOr("LISTEN_ADDR", ":8088")

	rateLimitDur = time.Hour

	// lgm1 + 38 bech32 characters.
	addrRe = regexp.MustCompile(`^lgm1[a-z0-9]{38}$`)

	// rateMu guards lastAccess.
	rateMu     sync.Mutex
	lastAccess = make(map[string]time.Time)

	// sendMu serializes transactions to avoid sequence number conflicts.
	sendMu sync.Mutex
)

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

type faucetReq struct {
	Address string `json:"address"`
}

type faucetResp struct {
	TxHash string `json:"tx_hash,omitempty"`
	Error  string `json:"error,omitempty"`
}

func handleFaucet(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}

	var req faucetReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, faucetResp{Error: "invalid JSON body"})
		return
	}

	if !addrRe.MatchString(req.Address) {
		writeJSON(w, http.StatusBadRequest, faucetResp{Error: "invalid address: must match lgm1..."})
		return
	}

	// Rate limit: 1 request per address per hour.
	rateMu.Lock()
	if last, ok := lastAccess[req.Address]; ok && time.Since(last) < rateLimitDur {
		rateMu.Unlock()
		remaining := rateLimitDur - time.Since(last)
		writeJSON(w, http.StatusTooManyRequests, faucetResp{
			Error: fmt.Sprintf("rate limited — try again in %s", remaining.Round(time.Second)),
		})
		return
	}
	lastAccess[req.Address] = time.Now()
	rateMu.Unlock()

	txHash, err := sendTokens(req.Address)
	if err != nil {
		// Roll back rate limit on failure so the user can retry.
		rateMu.Lock()
		delete(lastAccess, req.Address)
		rateMu.Unlock()
		log.Printf("ERROR send to %s: %v", req.Address, err)
		writeJSON(w, http.StatusInternalServerError, faucetResp{Error: err.Error()})
		return
	}

	log.Printf("OK send %s to %s tx=%s", faucetAmount, req.Address, txHash)
	writeJSON(w, http.StatusOK, faucetResp{TxHash: txHash})
}

func sendTokens(toAddr string) (string, error) {
	sendMu.Lock()
	defer sendMu.Unlock()

	cmd := exec.Command("legbled", "tx", "bank", "send",
		faucetKeyName, toAddr, faucetAmount,
		"--chain-id", chainID,
		"--node", nodeURL,
		"--keyring-backend", "test",
		"--home", keyringDir,
		"--gas", "auto",
		"--gas-adjustment", "1.5",
		"--gas-prices", "0.025ulgm",
		"-y", "--output", "json",
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("tx failed: %s (%s)", err, strings.TrimSpace(string(out)))
	}

	// CombinedOutput may mix stderr (gas estimate) with JSON — find first '{'.
	jsonData := extractJSON(out)

	var result struct {
		TxHash string `json:"txhash"`
		Code   int    `json:"code"`
		RawLog string `json:"raw_log"`
	}
	if err := json.Unmarshal(jsonData, &result); err != nil {
		return "", fmt.Errorf("parse response: %s (raw: %s)", err, strings.TrimSpace(string(out)))
	}
	if result.Code != 0 {
		return "", fmt.Errorf("tx rejected: code=%d %s", result.Code, result.RawLog)
	}

	return result.TxHash, nil
}

func extractJSON(data []byte) []byte {
	for i, b := range data {
		if b == '{' {
			return data[i:]
		}
	}
	return data
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// cleanupLoop removes stale rate-limit entries every hour.
func cleanupLoop() {
	for {
		time.Sleep(time.Hour)
		rateMu.Lock()
		now := time.Now()
		for addr, t := range lastAccess {
			if now.Sub(t) > rateLimitDur {
				delete(lastAccess, addr)
			}
		}
		rateMu.Unlock()
	}
}

const htmlPage = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>LGM Testnet Faucet</title>
<style>
  * { box-sizing: border-box; }
  body { font-family: system-ui, -apple-system, sans-serif; max-width: 480px; margin: 60px auto; padding: 0 20px; color: #1a202c; }
  h1 { margin-bottom: 4px; }
  p.sub { color: #718096; margin-top: 0; }
  input { width: 100%%; padding: 10px 12px; margin: 8px 0; font-size: 16px; border: 1px solid #cbd5e0; border-radius: 6px; }
  button { padding: 10px 24px; background: #3182ce; color: white; border: none; border-radius: 6px; cursor: pointer; font-size: 16px; }
  button:hover { background: #2c5282; }
  button:disabled { background: #a0aec0; cursor: not-allowed; }
  #result { margin-top: 16px; padding: 12px; border-radius: 6px; display: none; word-break: break-all; }
  .ok { background: #c6f6d5; color: #22543d; }
  .err { background: #fed7d7; color: #9b2c2c; }
</style>
</head>
<body>
<h1>LGM Testnet Faucet</h1>
<p class="sub">Get 10 LGM test tokens for the devnet.</p>
<form id="f">
  <input id="addr" placeholder="lgm1..." pattern="lgm1[a-z0-9]{38}" required />
  <button type="submit" id="btn">Request Tokens</button>
</form>
<div id="result"></div>
<script>
document.getElementById('f').onsubmit = async (e) => {
  e.preventDefault();
  const btn = document.getElementById('btn');
  const res = document.getElementById('result');
  btn.disabled = true;
  btn.textContent = 'Sending...';
  res.style.display = 'none';
  try {
    const r = await fetch('/faucet', {
      method: 'POST',
      headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({address: document.getElementById('addr').value})
    });
    const j = await r.json();
    res.style.display = 'block';
    if (j.error) { res.className = 'err'; res.textContent = j.error; }
    else { res.className = 'ok'; res.textContent = 'Sent! TX: ' + j.tx_hash; }
  } catch(err) {
    res.style.display = 'block';
    res.className = 'err';
    res.textContent = err.message;
  } finally {
    btn.disabled = false;
    btn.textContent = 'Request Tokens';
  }
};
</script>
</body>
</html>`

func main() {
	// If FAUCET_MNEMONIC is set, recover the key into the local keyring.
	if mnemonic := os.Getenv("FAUCET_MNEMONIC"); mnemonic != "" {
		log.Printf("Recovering faucet key %q from mnemonic...", faucetKeyName)

		// Ensure keyring directory exists.
		os.MkdirAll(keyringDir, 0o700)

		cmd := exec.Command("legbled", "keys", "add", faucetKeyName,
			"--recover",
			"--keyring-backend", "test",
			"--home", keyringDir,
		)
		cmd.Stdin = strings.NewReader(mnemonic + "\n")

		out, err := cmd.CombinedOutput()
		if err != nil {
			if strings.Contains(string(out), "already exists") {
				log.Printf("Key %q already in keyring, skipping", faucetKeyName)
			} else {
				log.Fatalf("Failed to recover faucet key: %s: %s", err, string(out))
			}
		} else {
			log.Println("Faucet key recovered successfully")
		}
	}

	go cleanupLoop()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, htmlPage)
	})
	http.HandleFunc("/faucet", handleFaucet)
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	})

	log.Printf("LGM Faucet listening on %s (chain=%s, node=%s, amount=%s)",
		listenAddr, chainID, nodeURL, faucetAmount)
	if err := http.ListenAndServe(listenAddr, nil); err != nil {
		log.Fatal(err)
	}
}

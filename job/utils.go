package job

import (
	"fmt"
	"net/http"
	"webhook/secret"
)

func CreateSecret(w http.ResponseWriter, r *http.Request) {
	body := `{"request_id":"test","message":"hello","callbackUrl":"http://localhost:8000/callback-check"}`

	sig := secret.CreateHMACSHA256(body)

	curl := fmt.Sprintf(`curl -i -X POST http://localhost:8000/hello \
-H "Content-Type: application/json" \
-H "X-Signature: %s" \
-d '%s'
`, sig, body)

	fmt.Print(curl)
	fmt.Fprint(w, curl)
}

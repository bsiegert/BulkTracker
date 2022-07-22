/*-
 * Copyright (c) 2022
 *      Benny Siegert <bsiegert@gmail.com>
 *
 * Provided that these terms and disclaimer and all copyright notices
 * are retained or reproduced in an accompanying document, permission
 * is granted to deal in this work without restriction, including un-
 * limited rights to use, publicly perform, distribute, sell, modify,
 * merge, give away, or sublicence.
 *
 * This work is provided "AS IS" and WITHOUT WARRANTY of any kind, to
 * the utmost extent permitted by applicable law, neither express nor
 * implied; without malicious intent or gross negligence. In no event
 * may a licensor, author or contributor be held liable for indirect,
 * direct, other damage, loss, or other issues arising in any way out
 * of dealing in the work, even if advised of the possibility of such
 * damage or existence of a defect, except proven that it results out
 * of said person's immediate fault when using the work as intended.
 */

// Binary standalone is the main entrypoint for BulkTracker as a stand-alone
// (non-App Engine) app.
package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"

	"github.com/bsiegert/BulkTracker/ingest"
	"github.com/bsiegert/BulkTracker/log"
)

var (
	port = flag.Int("port", 8080, "The port to use.")
)

func main() {
	flag.Parse()

	http.Handle("/", &ingest.IncomingMailHandler{})

	err := http.ListenAndServe(fmt.Sprintf(":%d", *port), nil)
	if err != nil {
		log.Errorf(context.Background(), "%s", err)
	}
}

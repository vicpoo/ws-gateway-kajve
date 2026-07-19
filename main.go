// WebSocket Gateway — kajve.
//
// Autentica al cliente por el mismo JWT que emite api-mobile, verifica que
// el lote solicitado le pertenezca, manda el histórico reciente de
// lecturas_ambientales para que la gráfica no arranque vacía, y luego
// reenvía en vivo cada evento que ingesta-iot publica en Redis (canal
// user:<id>).
package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"github.com/kajve/ws-gateway/src/infrastructure/dependencies"
	"github.com/kajve/ws-gateway/src/infrastructure/routes"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("main: no se encontró archivo .env, se usan las variables de entorno del sistema")
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	container, err := dependencies.NewContainer(ctx)
	if err != nil {
		log.Fatalf("main: error inicializando dependencias: %v", err)
	}
	defer container.Close()

	router := routes.NewRouter(container.WSHandler)
	server := &http.Server{
		Addr:              ":" + container.Config.Port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("main: WebSocket Gateway escuchando en :%s", container.Config.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("main: error en el servidor HTTP: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("main: señal de apagado recibida, cerrando...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("main: error cerrando el servidor HTTP: %v", err)
	}
	log.Println("main: apagado completo")
}

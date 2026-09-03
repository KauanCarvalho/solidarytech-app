package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

// otlpEndpoint resolve o endereço do OTel Collector, com fallback para o endereço interno do cluster.
func otlpEndpoint() string {
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		endpoint = "otel-collector.monitoring.svc.cluster.local:4317"
	}
	return endpoint
}

// newResource monta o Resource compartilhado por traces e métricas: nome/versão do serviço
// mais os atributos vindos de OTEL_RESOURCE_ATTRIBUTES (ex.: "namespace=donation-service"), usado
// pelas regras de alerta e pelo dashboard do Grafana para filtrar por serviço.
func newResource(ctx context.Context, serviceName string) (*resource.Resource, error) {
	return resource.New(ctx,
		resource.WithFromEnv(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
			semconv.ServiceVersionKey.String("1.0.0"),
		),
	)
}

// initTracer configura o exportador OTel gRPC e define o TracerProvider global.
func initTracer(serviceName string) (func(context.Context) error, error) {
	ctx := context.Background()
	endpoint := otlpEndpoint()

	log.Printf("Inicializando OTel Tracer para %s enviando para %s", serviceName, endpoint)

	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithTimeout(5*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("falha ao criar exportador trace OTLP: %w", err)
	}

	res, err := newResource(ctx, serviceName)
	if err != nil {
		return nil, fmt.Errorf("falha ao criar resource OTel: %w", err)
	}

	bsp := sdktrace.NewBatchSpanProcessor(exporter)
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return tp.Shutdown, nil
}

// initMeter configura o exportador OTel gRPC e define o MeterProvider global. Os instrumentos
// (contador/histograma) usados pelo middleware de métricas HTTP vivem em metrics.go.
func initMeter(serviceName string) (func(context.Context) error, error) {
	ctx := context.Background()
	endpoint := otlpEndpoint()

	log.Printf("Inicializando OTel Meter para %s enviando para %s", serviceName, endpoint)

	exporter, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithInsecure(),
		otlpmetricgrpc.WithEndpoint(endpoint),
		otlpmetricgrpc.WithTimeout(5*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("falha ao criar exportador metric OTLP: %w", err)
	}

	res, err := newResource(ctx, serviceName)
	if err != nil {
		return nil, fmt.Errorf("falha ao criar resource OTel: %w", err)
	}

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter, sdkmetric.WithInterval(15*time.Second))),
	)

	otel.SetMeterProvider(mp)

	return mp.Shutdown, nil
}

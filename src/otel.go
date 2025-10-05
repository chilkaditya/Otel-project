package main

import (
    "context"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/sdk/trace"
    "go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
    "go.opentelemetry.io/otel/sdk/metric"
    // "go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
    "go.opentelemetry.io/otel/sdk/log"
    "go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
)

var MetricReader *metric.ManualReader

func InitOpenTelemetry(ctx context.Context) error {
    // Tracing

    // Creates a stdout exporter (like Python’s ConsoleSpanExporter).
    traceExporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
    if err != nil {
        return err
    }

    // Creates a new TracerProvider.
    tracerProvider := trace.NewTracerProvider(trace.WithBatcher(traceExporter))

    //Registers your TracerProvider as the global default.
    otel.SetTracerProvider(tracerProvider)

    // Metrics
   MetricReader = metric.NewManualReader()
    meterProvider := metric.NewMeterProvider(
        metric.WithReader(MetricReader),
    )
    otel.SetMeterProvider(meterProvider)

    // // Logging
    logExporter, err := stdoutlog.New(stdoutlog.WithPrettyPrint())
    if err != nil {
        return err
    }
    logProvider := log.NewLoggerProvider(log.WithExporter(logExporter))
    otel.SetLoggerProvider(logProvider)

    return nil
}
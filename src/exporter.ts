interface HJDBResponse {
  state: "ok" | "err";
  data: PrometheusMetric[] | null;
  db: string | null;
  tab: string | null;
  err: string | null;
  store: "file" | "memory" | null;
}

interface PrometheusMetric {
  MetricName: string;
  MetricType: "gauge" | "counter";
  MetricValue: number;
}

async function fetchMetrics(): Promise<PrometheusMetric[]> {
  const response = await fetch(
    "http://192.168.200.146:8000/db/abcbcvb/tab/metrics_destination/store/memory",
  );
  if (response.ok) {
    const hjdbResp: HJDBResponse = await response.json();
    if (hjdbResp.state === "ok") {
      return hjdbResp.data ?? [];
    } else {
      throw new Error(hjdbResp.err ?? "");
    }
  } else {
    throw new Error("Failed to fetch metrics");
  }
}

function formatMetrics(metrics: PrometheusMetric[]): string {
  return metrics.map((metric) =>
    `# HELP ${metric.MetricName}\n# TYPE ${metric.MetricName} ${metric.MetricType}\n${metric.MetricName} ${metric.MetricValue}`
  ).join("\n") + "\n";
}

async function serveHandler(request: Request): Promise<Response> {
  const { pathname } = new URL(request.url);
  if (pathname === "/metrics") {
    try {
      const formattedMetrics = formatMetrics(await fetchMetrics());
      return new Response(formattedMetrics, { status: 200 });
    } catch (err) {
      console.error(err);
      return new Response(`Internal Server Error ${err}\n`, { status: 500 });
    }
  } else {
    return new Response("/metrics", { status: 404 });
  }
}

Deno.serve({ port: 8000 }, serveHandler);

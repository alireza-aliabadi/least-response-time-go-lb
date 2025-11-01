# Least Response Time Golang Load Balancer

## Usage
add your servers in **hosts** file.

run:
```shell
go run main.go
```

## Algorithm

**Exponentially Weighted Moving Average (EWMA)**: Instead of using just the last response time, we use a weighted average. This gives more weight to recent response times, allowing the system to adapt to changing conditions while smoothing out outliers. The formula is:

$NewAvg = (\alpha \times CurrentSample) + ((1-\alpha) \times OldAvg)$

Here, $\alpha$ is a smoothing factor between 0 and 1. A smaller $\alpha$ results in more smoothing. We'll use $\alpha=0.25$.

## Health Check
A background process (every 10 seconds) will periodically ping each backend server to ensure it's online. If a server goes down, it's temporarily removed from the pool of available servers.
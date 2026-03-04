setwd("//wsl.localhost/Ubuntu/home/nick898/repos/earth-discretization-benchmark")
library(ggplot2)

dfh3 = read.csv("output/h3-averages.csv")
dfs2 = read.csv("output/s2-averages.csv")
df = rbind(dfh3, dfs2)

dur = read.csv("output/durations-s2-res2.csv")
summary(unlist(dur$duration..ns.))

dur2 = read.csv("output/s2-caching-res2.csv")
summary(unlist(dur2$durationNs))

g = ggplot(df, aes(x = AvgAreaKm2, y = AverageDurationNs, color = Product)) +
  geom_line() +
  labs(title = "Uber H3 vs. Google S2 Polygon/Cell Intersection") + 
  xlab("Log of Average Cell Area (KM^2)") + 
  ylab("Log of Average Duration (nanoseconds)") + 
  scale_x_log10() + 
  scale_y_log10()
ggsave(filename = "output/h3-vs-s2.png", plot = g, width = 6, height = 4, dpi = 300)
# README

Assuming you are in the root of the repository

## Generate sample data
```
python3 data/generate_mock_polygons.py
```

## Benchmark 
```
go run .
```

## Example Output
```
H3 ================================================
Successfully converted 1 polygons to H3 GeoPolygon format

Processing Polygon 0
Duration (ns): 177785
Polygon 0 covers 11 H3 cells at resolution 3
  First few cell IDs: 8332d1fffffffff 8316d3fffffffff 8332d4fffffffff 
Durations: [177785]
S2 ================================================
Successfully converted 1 features to S2 regions

Feature 1 has 1 region(s)
Duration (ns): 13324138
Level: 10; Level Count: 2704
Level: 11; Level Count: 31
Level: 12; Level Count: 12
Level: 13; Level Count: 12
Level: 15; Level Count: 1
Level: 14; Level Count: 2
Durations: [13324138]
```

## Links

- https://geojson.io/
- https://vibhorsingh.com/boundingbox
- https://igorgatis.github.io/ws2/
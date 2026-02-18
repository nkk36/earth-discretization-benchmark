import random
import math
import json
from typing import List, Tuple, Dict, Any, Optional

def generate_random_polygon(
    min_size_km: float = 1,
    max_size_km: float = 5000,
    min_vertices: int = 4,
    max_vertices: int = 6,
    include_circles: bool = True,
    include_ovals: bool = True
) -> Tuple[List[List[List[float]]], str]:
    """
    Generate a random polygon of varying shapes on Earth in GeoJSON format.
    Ensures polygons do not wrap around the dateline or poles.
    
    Args:
        min_size_km: Minimum side length in kilometers (default: 1 km)
        max_size_km: Maximum side length in kilometers (default: 5000 km, roughly Africa's size)
        min_vertices: Minimum number of vertices for n-gons (default: 4)
        max_vertices: Maximum number of vertices for n-gons (default: 6)
        include_circles: Whether to include circles in the shape options (default: True)
        include_ovals: Whether to include ovals in the shape options (default: True)
    
    Returns:
        A tuple of (GeoJSON coordinate array, shape_type string)
    """
    
    # Keep trying until we get a valid polygon that doesn't wrap
    max_attempts = 100
    for attempt in range(max_attempts):
        # Random starting point (latitude, longitude)
        center_lat = random.uniform(-85, 85)
        center_lon = random.uniform(-180, 180)
        
        # Randomly generate size with equal distribution
        size_km = random.uniform(min_size_km, max_size_km)
        
        # Convert km to degrees (approximate)
        lat_radius_deg = size_km / 111.0
        lon_radius_deg = size_km / (111.0 * math.cos(math.radians(center_lat)))
        
        # Check if this configuration would wrap around poles or dateline
        if _would_wrap(center_lat, center_lon, lat_radius_deg, lon_radius_deg):
            continue
        
        # Build list of available shape types
        shape_types = []
        
        # Always include n-gons with vertices from min_vertices to max_vertices
        for num_vertices in range(min_vertices, max_vertices + 1):
            shape_types.append(('ngon', num_vertices))
        
        # Conditionally add circles and ovals
        if include_circles:
            shape_types.append(('circle', None))
        if include_ovals:
            shape_types.append(('oval', None))
        
        # Randomly choose a shape type
        shape_choice = random.choice(shape_types)
        shape_type_name = shape_choice[0]
        shape_param = shape_choice[1]
        
        if shape_type_name == 'ngon':
            coordinates = _create_ngon(center_lat, center_lon, lat_radius_deg, lon_radius_deg, sides=shape_param)
            shape_type = f"{shape_param}-gon"
        elif shape_type_name == 'circle':
            coordinates = _create_circle(center_lat, center_lon, lat_radius_deg, lon_radius_deg)
            shape_type = 'circle'
        elif shape_type_name == 'oval':
            coordinates = _create_oval(center_lat, center_lon, lat_radius_deg, lon_radius_deg)
            shape_type = 'oval'
        
        return [coordinates], shape_type
    
    # Fallback (shouldn't normally reach here)
    raise RuntimeError(f"Could not generate valid polygon after {max_attempts} attempts")


def _would_wrap(center_lat: float, center_lon: float, lat_radius: float, lon_radius: float) -> bool:
    """
    Check if a polygon would wrap around the poles or dateline.
    
    Args:
        center_lat: Center latitude
        center_lon: Center longitude
        lat_radius: Latitude radius in degrees
        lon_radius: Longitude radius in degrees
    
    Returns:
        True if polygon would wrap, False otherwise
    """
    # Check if it would exceed pole boundaries
    if center_lat + lat_radius > 85 or center_lat - lat_radius < -85:
        return True
    
    # Check if it would cross the dateline (longitude wrapping)
    # We check if the east or west edge would cross ±180
    if center_lon + lon_radius > 180 or center_lon - lon_radius < -180:
        return True
    
    return False


def _create_ngon(center_lat: float, center_lon: float, lat_radius: float, lon_radius: float, sides: int) -> List[List[float]]:
    """Create a regular n-sided polygon."""
    coordinates = []
    
    for i in range(sides):
        angle = (2 * math.pi * i) / sides
        
        # Calculate point on ellipse
        x = lon_radius * math.cos(angle)
        y = lat_radius * math.sin(angle)
        
        lat = center_lat + y
        lon = center_lon + x
        
        coordinates.append([lon, lat])
    
    # Close the polygon
    coordinates.append(coordinates[0])
    
    return coordinates


def _create_circle(center_lat: float, center_lon: float, lat_radius: float, lon_radius: float) -> List[List[float]]:
    """Create a circular polygon (approximated with many points)."""
    coordinates = []
    num_points = 64  # Number of points to approximate the circle
    
    for i in range(num_points):
        angle = (2 * math.pi * i) / num_points
        
        # Use average of lat/lon radius for more circular appearance
        avg_radius = (lat_radius + lon_radius) / 2
        x = avg_radius * math.cos(angle)
        y = avg_radius * math.sin(angle)
        
        lat = center_lat + y
        lon = center_lon + x
        
        coordinates.append([lon, lat])
    
    # Close the polygon
    coordinates.append(coordinates[0])
    
    return coordinates


def _create_oval(center_lat: float, center_lon: float, lat_radius: float, lon_radius: float) -> List[List[float]]:
    """Create an oval/ellipse polygon."""
    coordinates = []
    num_points = 64  # Number of points to approximate the oval
    
    # Randomly choose orientation (0 = stretched east-west, 1 = stretched north-south)
    orientation = random.choice([0, 1])
    
    if orientation == 0:
        # Stretched east-west
        major_radius = lon_radius * 1.5
        minor_radius = lat_radius * 0.7
    else:
        # Stretched north-south
        major_radius = lat_radius * 1.5
        minor_radius = lon_radius * 0.7
    
    for i in range(num_points):
        angle = (2 * math.pi * i) / num_points
        
        if orientation == 0:
            x = major_radius * math.cos(angle)
            y = minor_radius * math.sin(angle)
        else:
            x = minor_radius * math.cos(angle)
            y = major_radius * math.sin(angle)
        
        lat = center_lat + y
        lon = center_lon + x
        
        coordinates.append([lon, lat])
    
    # Close the polygon
    coordinates.append(coordinates[0])
    
    return coordinates


def generate_mock_polygons(
    count: int,
    min_size_km: float = 1,
    max_size_km: float = 5000,
    min_vertices: int = 4,
    max_vertices: int = 6,
    include_circles: bool = True,
    include_ovals: bool = True
) -> Dict[str, Any]:
    """
    Generate multiple mock polygons of varying shapes as a GeoJSON FeatureCollection.
    Ensures polygons do not wrap around the dateline or poles.
    
    Args:
        count: Number of polygons to generate
        min_size_km: Minimum side length in kilometers
        max_size_km: Maximum side length in kilometers
        min_vertices: Minimum number of vertices for n-gons (default: 4)
        max_vertices: Maximum number of vertices for n-gons (default: 6)
        include_circles: Whether to include circles in the shape options (default: True)
        include_ovals: Whether to include ovals in the shape options (default: True)
    
    Returns:
        A GeoJSON FeatureCollection dictionary
    """
    features = []
    
    for i in range(count):
        coordinates, shape_type = generate_random_polygon(
            min_size_km=min_size_km,
            max_size_km=max_size_km,
            min_vertices=min_vertices,
            max_vertices=max_vertices,
            include_circles=include_circles,
            include_ovals=include_ovals
        )
        
        feature = {
            "type": "Feature",
            "geometry": {
                "type": "Polygon",
                "coordinates": coordinates
            },
            "properties": {
                "id": i + 1,
                "shape": shape_type
            }
        }
        features.append(feature)
    
    return {
        "type": "FeatureCollection",
        "features": features
    }


def print_geojson(geojson: Dict[str, Any]) -> None:
    """Pretty print the GeoJSON."""
    print(json.dumps(geojson, indent=2))


if __name__ == "__main__":
    # # Example 1: Default (4-6 sided polygons, circles, and ovals)
    # print("=" * 60)
    # print("Example 1: Default settings (4-6 vertices, circles, ovals)")
    # print("=" * 60)
    # geojson = generate_mock_polygons(count=5)
    # print_geojson(geojson)
    
    # with open("mock_polygons_default.geojson", "w") as f:
    #     json.dump(geojson, f, indent=2)
    # print("\nExported to 'mock_polygons_default.geojson'\n")
    
    # # Example 2: Only rectangles and pentagons, no circles or ovals
    # print("=" * 60)
    # print("Example 2: Rectangles and pentagons only (no circles/ovals)")
    # print("=" * 60)
    # geojson = generate_mock_polygons(
    #     count=5,
    #     min_vertices=4,
    #     max_vertices=5,
    #     include_circles=False,
    #     include_ovals=False
    # )
    # print_geojson(geojson)
    
    # with open("mock_polygons_no_curves.geojson", "w") as f:
    #     json.dump(geojson, f, indent=2)
    # print("\nExported to 'mock_polygons_no_curves.geojson'\n")
    
    # # Example 3: High vertex count (4-12 sided), circles and ovals only
    # print("=" * 60)
    # print("Example 3: Complex polygons (4-12 vertices, circles, ovals)")
    # print("=" * 60)
    # geojson = generate_mock_polygons(
    #     count=5,
    #     min_vertices=4,
    #     max_vertices=12,
    #     include_circles=True,
    #     include_ovals=True
    # )
    # print_geojson(geojson)
    
    # with open("mock_polygons_complex.geojson", "w") as f:
    #     json.dump(geojson, f, indent=2)
    # print("\nExported to 'mock_polygons_complex.geojson'\n")

    # Inputs for generating sample polygons
    num_polygons = 1000
    min_size_km = 1
    max_size_km = 315
    min_vertices = 4
    max_vertices = 4
    circles = True
    ovals = True
        
    # Generate samples
    geojson = generate_mock_polygons(
        count=num_polygons, 
        min_size_km=min_size_km, 
        max_size_km=max_size_km,
        min_vertices = min_vertices, 
        max_vertices = max_vertices,
        include_circles = circles, 
        include_ovals = ovals
        )
    
    print(f"Generated {num_polygons} random polygons (GeoJSON format):\n")
        
    # Export as JSON
    with open("data/mock_polygons.geojson", "w") as f:
        json.dump(geojson, f, indent=2)
    print(f"\n\nPolygons exported to 'mock_polygons.geojson'")
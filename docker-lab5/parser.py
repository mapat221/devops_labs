import time
import requests
from prometheus_client import start_http_server, Gauge

ISS_VELOCITY = Gauge('iss_velocity', 'Current velocity of the ISS in km/h')
ISS_ALTITUDE = Gauge('iss_altitude', 'Current altitude of the ISS in km')

def get_iss_data():
    try:
        response = requests.get("https://api.wheretheiss.at/v1/satellites/25544", timeout=30)
        data = response.json()
        velocity = data['velocity']
        altitude = data['altitude']
        ISS_VELOCITY.set(velocity)
        ISS_ALTITUDE.set(altitude)
        print(f"fetched data. Velocity: {velocity}, altitude: {altitude}")
    except Exception as e:
        print(f"Request error: {e}")

if __name__ == '__main__':
    start_http_server(8000)
    while True:
        get_iss_data()
        time.sleep(15)
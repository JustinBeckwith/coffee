function initialize() {
  var mapOptions = {
    center: {
      lat: 47.649038,
      lng: -122.3502358
    },
    zoom: 16
  };
  var map = new google.maps.Map(document.getElementById('map-canvas'), mapOptions);

  navigator.geolocation.getCurrentPosition(function(position) {
    getCoffee(position);
  });
}

function getCoffee(position) {
  fetch('/GetCoffee?lat=' + position.coords.latitude + "&lon=" + position.coords.longitude)
  .then(function(response) {
    return response.json()
  }).then(function(json) {
    console.log('parsed json', json)
  }).catch(function(ex) {
    console.log('parsing failed', ex)
  });
}

google.maps.event.addDomListener(window, 'load', initialize);

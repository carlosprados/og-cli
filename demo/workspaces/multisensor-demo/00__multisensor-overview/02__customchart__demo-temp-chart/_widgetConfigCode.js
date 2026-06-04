// Current environment values per device — fetched with $api (opengate-js),
// the blessed data API for widget JS (the http wrapper 403s on north endpoints).
// IMPORTANT: the platform lints widget code with a strict JSHint at render
// time — stick to ES5 style (var, function, forEach; no const/arrows/for-of).
// A single top-level await is accepted (the code runs inside an async wrapper).
var devices = ['multisensor-001', 'multisensor-002', 'multisensor-003'];
var streams = [
  { id: 'sensor.temperature', label: 'Temperature (ºC)' },
  { id: 'sensor.humidity', label: 'Humidity (%)' },
  { id: 'power.battery', label: 'Battery (%)' }
];

function lastValue(deviceId, datastreamId) {
  // .filter() with explicit field names — the builder's withDeviceId/
  // withDatastream shortcuts emit outdated field names (datapoint.device)
  // that the platform rejects with "Field in filter unknown".
  return $api.datapointsSearchBuilder()
    .filter({
      and: [
        { eq: { 'datapoints.entityIdentifier': deviceId } },
        { eq: { 'datapoints.datastreamId': datastreamId } }
      ]
    })
    .build()
    .execute()
    .then(function (res) {
      var dps = (res && res.data && res.data.datapoints) ? res.data.datapoints : [];
      var latest = null;
      dps.forEach(function (dp) {
        var c = dp._current || {};
        if (c.value !== null && c.value !== undefined) {
          if (latest === null || new Date(c.at) > new Date(latest.at)) {
            latest = c;
          }
        }
      });
      return latest ? latest.value : null;
    })
    .catch(function (e) {
      console.error('datapoints error', deviceId, datastreamId, e);
      return null;
    });
}

var calls = [];
devices.forEach(function (dev) {
  streams.forEach(function (st) {
    calls.push(lastValue(dev, st.id));
  });
});

var flat = await Promise.all(calls);

var series = streams.map(function (st, sIdx) {
  return {
    name: st.label,
    type: 'bar',
    data: devices.map(function (ignored, dIdx) {
      return flat[dIdx * streams.length + sIdx];
    })
  };
});

return {
  tooltip: { trigger: 'axis' },
  legend: {
    data: streams.map(function (st) { return st.label; })
  },
  xAxis: { type: 'category', data: devices },
  yAxis: { type: 'value' },
  series: series
};

// Multi-sensor summary table: one row per device, a sparkline per datastream.
// Rewritten 2026-06-04 to the verified widget-JS style (see the og-cli repo
// grimoire: .claude/skills/og-workspaces/reference/widget-js-api.md):
// ES5 only (platform JSHint) + $api with explicit .filter() field names.
var devices = [
  { id: 'multisensor-001', name: 'Multi-Sensor 001' },
  { id: 'multisensor-002', name: 'Multi-Sensor 002' },
  { id: 'multisensor-003', name: 'Multi-Sensor 003' }
];
var streams = [
  { id: 'sensor.temperature', field: 'temperature', unit: 'ºC',  color: '#e6550d' },
  { id: 'sensor.humidity',    field: 'humidity',    unit: '%',   color: '#3182bd' },
  { id: 'sensor.luminosity',  field: 'luminosity',  unit: 'lx',  color: '#fd8d3c' },
  { id: 'energy.consumption', field: 'energy',      unit: 'kWh', color: '#31a354' },
  { id: 'power.battery',      field: 'battery',     unit: '%',   color: '#756bb1' }
];

function sparkline(points, color, unit) {
  var times = points.map(function (p) { return p[0]; });
  var values = points.map(function (p) { return p[1]; });
  var last = values.length ? values[values.length - 1] : null;
  return {
    grid: { left: 6, right: 6, top: 22, bottom: 6 },
    tooltip: {
      trigger: 'axis',
      formatter: function (ps) {
        var p = ps[0];
        var when = $moment ? $moment(p.axisValue).format('DD/MM HH:mm') : p.axisValue;
        return when + '<br/>' + p.data + ' ' + unit;
      }
    },
    xAxis: { type: 'category', data: times, show: false },
    yAxis: { type: 'value', scale: true, show: false },
    title: {
      text: (last !== null ? last + ' ' + unit : 'n/a'),
      right: 6, top: 0,
      textStyle: { fontSize: 12, fontWeight: 'bold', color: color }
    },
    series: [{
      type: 'line', data: values, smooth: true, showSymbol: false,
      lineStyle: { color: color, width: 2 },
      areaStyle: { color: color, opacity: 0.15 }
    }]
  };
}

function loadStream(deviceId, datastreamId) {
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
      var pts = [];
      dps.forEach(function (dp) {
        var c = dp._current || {};
        if (c.at !== null && c.at !== undefined && c.value !== null && c.value !== undefined) {
          pts.push([c.at, c.value]);
        }
      });
      pts.sort(function (a, b) { return new Date(a[0]) - new Date(b[0]); });
      return pts;
    })
    .catch(function (e) {
      console.error('datapoints error', deviceId, datastreamId, e);
      return null;
    });
}

var calls = [];
devices.forEach(function (dev) {
  streams.forEach(function (st) {
    calls.push(loadStream(dev.id, st.id));
  });
});

// customTable contract: RETURN the rows (or a Promise of them); callback()
// is also invoked for executor variants that rely on it. Avoid top-level
// await here — the table executor calls .then() on the returned value.
return Promise.all(calls).then(function (flat) {
  var rows = devices.map(function (dev, dIdx) {
    var row = { id: dev.id, name: dev.name };
    streams.forEach(function (st, sIdx) {
      var pts = flat[dIdx * streams.length + sIdx];
      if (pts === null) {
        row[st.field] = { value: 'error', _style: 'color:#d62728;text-align:center;' };
      } else if (pts.length === 0) {
        row[st.field] = { value: 'no data', _style: 'color:#999;text-align:center;' };
      } else {
        row[st.field] = { value: '', _chart: sparkline(pts, st.color, st.unit) };
      }
    });
    return row;
  });
  if (typeof callback === 'function') { callback(rows); }
  return rows;
});

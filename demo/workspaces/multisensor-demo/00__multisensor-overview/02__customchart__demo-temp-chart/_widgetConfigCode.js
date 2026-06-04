// Current environment values per device — fetched live from the North API.
// Context: async function(entityData, relatedEntities, timeserieData, alarmData, dashboardFilters, filters, callback)
var body = {
  filter: { like: { 'provision.device.identifier': 'multisensor' } },
  select: [
    { name: 'provision.device.identifier', fields: [{ field: 'value', alias: 'id' }] },
    { name: 'sensor.temperature', fields: [{ field: 'value', alias: 'temp' }] },
    { name: 'sensor.humidity', fields: [{ field: 'value', alias: 'hum' }] },
    { name: 'power.battery', fields: [{ field: 'value', alias: 'batt' }] }
  ]
};

function pick(dev, field) {
  var f = dev[field];
  if (f && f._value && f._value._current && typeof f._value._current.value === 'number') {
    return f._value._current.value;
  }
  return null;
}

try {
  var res = await http.post('/north/v80/search/devices?flattened=true', body);
  var devices = (res && res.data && res.data.devices) ? res.data.devices : (res && res.devices ? res.devices : []);
  var names = [], temps = [], hums = [], batts = [];
  devices.forEach(function (dev) {
    var idField = dev['provision.device.identifier'];
    var id = (idField && idField._value && idField._value._current) ? idField._value._current.value : '?';
    names.push(id);
    temps.push(pick(dev, 'sensor.temperature'));
    hums.push(pick(dev, 'sensor.humidity'));
    batts.push(pick(dev, 'power.battery'));
  });

  return {
    tooltip: { trigger: 'axis' },
    legend: { data: ['Temperature (ºC)', 'Humidity (%)', 'Battery (%)'] },
    xAxis: { type: 'category', data: names },
    yAxis: { type: 'value' },
    series: [
      { name: 'Temperature (ºC)', type: 'bar', data: temps },
      { name: 'Humidity (%)', type: 'bar', data: hums },
      { name: 'Battery (%)', type: 'bar', data: batts }
    ]
  };
} catch (e) {
  console.log('demo chart fallback:', e);
  return {
    title: { text: 'Could not load live data', subtext: String(e), left: 'center' },
    series: []
  };
}

// Environmental anomaly — ADVANCED rule (edited locally as a real .js file).
//
// Opens an URGENT alarm when the device is HOT and DRY at the same time:
//   temperature > tempThreshold  AND  humidity < minHumidity
//
// Rising-edge detection (hysteresis): the alarm opens only when temperature
// CROSSES the threshold, not on every datapoint above it — multi-datastream,
// stateful logic that EASY mode cannot express.
//
// Rule JS context: entity['<datastream>']._value._current / ._previous,
// parameterObject + getVariableValue(), ruleName, openAlarm().

var tempThreshold = getVariableValue(parameterObject['tempThreshold']);
var minHumidity = getVariableValue(parameterObject['minHumidity']);

var t = entity['sensor.temperature'];
var h = entity['sensor.humidity'];

var temp = (t && t._value && t._value._current) ? t._value._current.value : null;
var prevTemp = (t && t._value && t._value._previous) ? t._value._previous.value : null;
var humidity = (h && h._value && h._value._current) ? h._value._current.value : null;

if (temp !== null && humidity !== null && temp > tempThreshold && humidity < minHumidity) {
    // Only on the rising edge: previous sample was at or below the threshold
    if (prevTemp === null || prevTemp <= tempThreshold) {
        openAlarm(null, 'Environmental anomaly', ruleName, 'URGENT', 'HIGH',
            'Hot & dry anomaly: ' + temp + ' ºC with ' + humidity + '% relative humidity');
    }
}

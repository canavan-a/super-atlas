/**
 * @format
 */

import { AppRegistry } from 'react-native';
import App from './App';
import { name as appName } from './app.json';
// Defines the background location task at module top level, so it exists
// when Android starts the app headless just to run it.
import './src/location/backgroundTask';

AppRegistry.registerComponent(appName, () => App);

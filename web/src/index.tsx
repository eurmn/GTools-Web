/* @refresh reload */
import { render } from 'solid-js/web';

import './index.css';
import App from './App';
import "@fontsource/inter";
import "@fontsource/quicksand";

render(() => <App />, document.getElementById('root') as HTMLElement);

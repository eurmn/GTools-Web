/* @refresh reload */
import { render } from 'solid-js/web';

import './index.css';
import App from './App';
import "@fontsource/inter/variable.css";
import "@fontsource/quicksand/variable.css";

render(() => <App />, document.getElementById('root') as HTMLElement);

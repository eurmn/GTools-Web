/* @refresh reload */
import { render } from 'solid-js/web';

import 'virtual:windi.css'
import './index.css';
import App from './App';
import "@fontsource/inter/variable.css";
import "@fontsource/quicksand/variable.css";
import "@fontsource/ubuntu-mono";

render(() => <App />, document.getElementById('root') as HTMLElement);

import { DoBootstrap, Injector, NgModule, provideBrowserGlobalErrorListeners } from '@angular/core';
import { BrowserModule } from '@angular/platform-browser';
import { createCustomElement } from '@angular/elements';

import { App } from './app';

@NgModule({
  imports: [
    BrowserModule,
    App
  ],
  providers: [
    provideBrowserGlobalErrorListeners()
  ]
})
export class AppModule implements DoBootstrap {
  private readonly injector: Injector;

  constructor(injector: Injector) {
    this.injector = injector;
  }

  ngDoBootstrap(): void {
    const el = createCustomElement(App, { injector: this.injector });
    customElements.define('core-element-template', el);
  }
}

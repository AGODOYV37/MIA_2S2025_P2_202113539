import { ComponentFixture, TestBed } from '@angular/core/testing';

import { Consola } from './consola';

describe('Consola', () => {
  let component: Consola;
  let fixture: ComponentFixture<Consola>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [Consola]
    })
    .compileComponents();

    fixture = TestBed.createComponent(Consola);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
